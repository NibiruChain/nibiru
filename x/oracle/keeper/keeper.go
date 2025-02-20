package keeper

import (
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// Keeper of the oracle store
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	AccountKeeper  types.AccountKeeper
	bankKeeper     types.BankKeeper
	distrKeeper    types.DistributionKeeper
	StakingKeeper  types.StakingKeeper
	slashingKeeper types.SlashingKeeper
	sudoKeeper     types.SudoKeeper

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
	storeKey storetypes.StoreKey,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,
	slashingKeeper types.SlashingKeeper,
	sudoKeeper types.SudoKeeper,

	distrName string,
) Keeper {
	// ensure oracle module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	k := Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		AccountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		distrKeeper:       distrKeeper,
		StakingKeeper:     stakingKeeper,
		slashingKeeper:    slashingKeeper,
		sudoKeeper:        sudoKeeper,
		distrModuleName:   distrName,
		Params:            collections.NewItem(storeKey, 11, collections.ProtoValueEncoder[types.Params](cdc)),
		ExchangeRates:     collections.NewMap(storeKey, 1, asset.PairKeyEncoder, collections.ProtoValueEncoder[types.DatedPrice](cdc)),
		PriceSnapshots:    collections.NewMap(storeKey, 10, collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder), collections.ProtoValueEncoder[types.PriceSnapshot](cdc)),
		FeederDelegations: collections.NewMap(storeKey, 2, collections.ValAddressKeyEncoder, collections.AccAddressValueEncoder),
		MissCounters:      collections.NewMap(storeKey, 3, collections.ValAddressKeyEncoder, collections.Uint64ValueEncoder),
		Prevotes:          collections.NewMap(storeKey, 4, collections.ValAddressKeyEncoder, collections.ProtoValueEncoder[types.AggregateExchangeRatePrevote](cdc)),
		Votes:             collections.NewMap(storeKey, 5, collections.ValAddressKeyEncoder, collections.ProtoValueEncoder[types.AggregateExchangeRateVote](cdc)),
		WhitelistedPairs:  collections.NewKeySet(storeKey, 6, asset.PairKeyEncoder),
		Rewards: collections.NewMap(
			storeKey, 7,
			collections.Uint64KeyEncoder, collections.ProtoValueEncoder[types.Rewards](cdc)),
		RewardsID: collections.NewSequence(storeKey, 9),
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
		delegate := k.FeederDelegations.GetOr(
			ctx, validatorAddr, sdk.AccAddress(validatorAddr))
		if !delegate.Equals(feederAddr) {
			return sdkerrors.Wrapf(
				types.ErrNoVotingPermission,
				"wanted: %s, got: %s", delegate.String(), feederAddr.String())
		}
	}

	// Check that the given validator is in the active set for consensus.
	if val := k.StakingKeeper.Validator(ctx, validatorAddr); val == nil || !val.IsBonded() {
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

	snapshots := k.PriceSnapshots.Iterate(
		ctx,
		collections.PairRange[asset.Pair, time.Time]{}.
			Prefix(pair).
			StartInclusive(
				ctx.BlockTime().Add(-1*params.TwapLookbackWindow)).
			EndInclusive(
				ctx.BlockTime()),
	).Values()

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
	k.ExchangeRates.Insert(ctx, pair, types.DatedPrice{ExchangeRate: price, CreatedBlock: uint64(ctx.BlockHeight())})

	key := collections.Join(pair, ctx.BlockTime())
	timestampMs := ctx.BlockTime().UnixMilli()
	k.PriceSnapshots.Insert(ctx, key, types.PriceSnapshot{
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
