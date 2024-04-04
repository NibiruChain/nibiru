package keeper

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/cosmos/cosmos-sdk/runtime"
	"time"

	storetypes "cosmossdk.io/store/types"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// Keeper of the oracle store
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	AccountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	distrKeeper   types.DistributionKeeper
	StakingKeeper types.StakingKeeper
	SudoKeeper    types.SudoKeeper

	distrModuleName string

	// Module parameters
	Params            collections.Item[types.Params]
	ExchangeRates     collections.Map[asset.Pair, types.DatedPrice]
	FeederDelegations collections.Map[sdk.ValAddress, sdk.AccAddress]
	MissCounters      collections.Map[sdk.ValAddress, uint64]
	Prevotes          collections.Map[sdk.ValAddress, types.AggregateExchangeRatePrevote]
	Votes             collections.Map[sdk.ValAddress, types.AggregateExchangeRateVote]

	// PriceSnapshots maps types.PriceSnapshot to the asset.Pair of the snapshot and the creation timestamp as keys.Uint64Key.
	PriceSnapshots collections.Map[
		collections.Pair[asset.Pair, time.Time],
		types.PriceSnapshot]
	WhitelistedPairs collections.KeySet[asset.Pair]
	Rewards          collections.Map[uint64, types.Rewards]
	RewardsID        collections.Sequence
}

// NewKeeper constructs a new keeper for oracle
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey *storetypes.KVStoreKey,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,
	sudoKeeper types.SudoKeeper,

	distrName string,
) Keeper {
	// ensure oracle module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	storeService := runtime.NewKVStoreService(storeKey)
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		AccountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		distrKeeper:       distrKeeper,
		StakingKeeper:     stakingKeeper,
		SudoKeeper:        sudoKeeper,
		distrModuleName:   distrName,
		Params:            collections.NewItem(sb, collections.NewPrefix(11), storeKey.String(), codec.CollValue[types.Params](cdc)),
		ExchangeRates:     collections.NewMap(sb, collections.NewPrefix(1), storeKey.String(), asset.PairKeyEncoder, codec.CollValue[types.DatedPrice](cdc)),
		PriceSnapshots:    collections.NewMap(sb, collections.NewPrefix(10), storeKey.String(), collections.PairKeyCodec(asset.PairKeyEncoder, sdk.TimeKey), codec.CollValue[types.PriceSnapshot](cdc)),
		FeederDelegations: collections.NewMap(sb, collections.NewPrefix(2), storeKey.String(), sdk.ValAddressKey, common.AccAddressValue),
		MissCounters:      collections.NewMap(sb, collections.NewPrefix(3), storeKey.String(), sdk.ValAddressKey, collections.Uint64Value),
		Prevotes:          collections.NewMap(sb, collections.NewPrefix(4), storeKey.String(), sdk.ValAddressKey, codec.CollValue[types.AggregateExchangeRatePrevote](cdc)),
		Votes:             collections.NewMap(sb, collections.NewPrefix(5), storeKey.String(), sdk.ValAddressKey, codec.CollValue[types.AggregateExchangeRateVote](cdc)),
		WhitelistedPairs:  collections.NewKeySet(sb, collections.NewPrefix(6), storeKey.String(), asset.PairKeyEncoder),
		Rewards: collections.NewMap(
			sb,
			collections.NewPrefix(7),
			storeKey.String(),
			collections.Uint64Key,
			codec.CollValue[types.Rewards](cdc),
		),
		RewardsID: collections.NewSequence(sb, collections.NewPrefix(9), storeKey.String()),
	}
	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ValidateFeeder return the given feeder is allowed to feed the message or not
func (k Keeper) ValidateFeeder(
	ctx sdk.Context, feederAddr sdk.AccAddress, validatorAddr sdk.ValAddress,
) error {
	// A validator delegates price feeder consent to itself by default.
	// Thus, we only need to verify consent for price feeder addresses that don't
	// match the validator address.
	if !feederAddr.Equals(validatorAddr) {
		delegate, err := k.FeederDelegations.Get(ctx, validatorAddr)
		if err != nil {
			delegate = sdk.AccAddress(validatorAddr)
		}
		if !delegate.Equals(feederAddr) {
			return sdkerrors.Wrapf(
				types.ErrNoVotingPermission,
				"wanted: %s, got: %s", delegate.String(), feederAddr.String())
		}
	}

	// Check that the given validator is in the active set for consensus.
	val, _ := k.StakingKeeper.Validator(ctx, validatorAddr)

	if val == nil || !val.IsBonded() {
		return sdkerrors.Wrapf(
			stakingtypes.ErrNoValidatorFound,
			"validator %s is not active set", validatorAddr.String())
	}

	return nil
}

func (k Keeper) GetExchangeRateTwap(ctx sdk.Context, pair asset.Pair) (price math.LegacyDec, err error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.LegacyOneDec().Neg(), err
	}

	iter, err := k.PriceSnapshots.Iterate(
		ctx,
		collections.NewPrefixedPairRange[asset.Pair, time.Time](pair).
			StartInclusive(
				ctx.BlockTime().Add(-1*params.TwapLookbackWindow)).
			EndInclusive(
				ctx.BlockTime()),
	)
	if err != nil {
		return math.LegacyZeroDec(), err
	}
	snapshots, err := iter.Values()
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	if len(snapshots) == 0 {
		// if there are no snapshots, return -1 for the price
		return math.LegacyOneDec().Neg(), types.ErrNoValidTWAP.Wrapf("no snapshots for pair %s", pair.String())
	}

	if len(snapshots) == 1 {
		return snapshots[0].Price, nil
	}

	firstTimestampMs := snapshots[0].TimestampMs
	if firstTimestampMs > ctx.BlockTime().UnixMilli() {
		// should never happen, or else we have corrupted state
		return math.LegacyOneDec().Neg(), types.ErrNoValidTWAP.Wrapf(
			"Possible corrupted state. First timestamp %d is after current blocktime %d", firstTimestampMs, ctx.BlockTime().UnixMilli())
	}

	if firstTimestampMs == ctx.BlockTime().UnixMilli() {
		// shouldn't happen because we check for len(snapshots) == 1, but if it does, return the first snapshot price
		return snapshots[0].Price, nil
	}

	cumulativePrice := math.LegacyZeroDec()
	for i, s := range snapshots {
		var nextTimestampMs int64
		if i == len(snapshots)-1 {
			// if we're at the last snapshot, then consider that price as ongoing until the current blocktime
			nextTimestampMs = ctx.BlockTime().UnixMilli()
		} else {
			nextTimestampMs = snapshots[i+1].TimestampMs
		}

		price := s.Price.MulInt64(nextTimestampMs - s.TimestampMs)
		cumulativePrice = cumulativePrice.Add(price)
	}

	return cumulativePrice.QuoInt64(ctx.BlockTime().UnixMilli() - firstTimestampMs), nil
}

func (k Keeper) GetExchangeRate(ctx sdk.Context, pair asset.Pair) (price math.LegacyDec, err error) {
	exchangeRate, err := k.ExchangeRates.Get(ctx, pair)
	price = exchangeRate.ExchangeRate
	return
}

// SetPrice sets the price for a pair as well as the price snapshot.
func (k Keeper) SetPrice(ctx sdk.Context, pair asset.Pair, price math.LegacyDec) {
	k.ExchangeRates.Set(ctx, pair, types.DatedPrice{ExchangeRate: price, CreatedBlock: uint64(ctx.BlockHeight())})

	key := collections.Join(pair, ctx.BlockTime())
	timestampMs := ctx.BlockTime().UnixMilli()
	k.PriceSnapshots.Set(ctx, key, types.PriceSnapshot{
		Pair:        pair,
		Price:       price,
		TimestampMs: timestampMs,
	})
	if err := ctx.EventManager().EmitTypedEvent(&types.EventPriceUpdate{
		Pair:        pair.String(),
		Price:       price,
		TimestampMs: timestampMs,
	}); err != nil {
		ctx.Logger().Error("failed to emit OraclePriceUpdate", "pair", pair, "error", err)
	}
}
