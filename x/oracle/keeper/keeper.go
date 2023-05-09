package keeper

import (
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/math"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// Keeper of the oracle store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   sdk.StoreKey
	paramSpace paramstypes.Subspace

	AccountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	distrKeeper   types.DistributionKeeper
	StakingKeeper types.StakingKeeper

	distrModuleName string

	ExchangeRates     collections.Map[asset.Pair, sdk.Dec]
	FeederDelegations collections.Map[sdk.ValAddress, sdk.AccAddress]
	MissCounters      collections.Map[sdk.ValAddress, uint64]
	Prevotes          collections.Map[sdk.ValAddress, types.AggregateExchangeRatePrevote]
	Votes             collections.Map[sdk.ValAddress, types.AggregateExchangeRateVote]

	// PriceSnapshots maps types.PriceSnapshot to the asset.Pair of the snapshot and the creation timestamp as keys.Uint64Key.
	PriceSnapshots   collections.Map[collections.Pair[asset.Pair, time.Time], types.PriceSnapshot]
	WhitelistedPairs collections.KeySet[asset.Pair]
	PairRewards      collections.Map[uint64, types.PairReward]
	PairRewardsID    collections.Sequence
}

type PairRewardsIndexes struct {
	// RewardsByPair is the index that maps rewards associated with specific pairs.
	RewardsByPair collections.MultiIndex[asset.Pair, uint64, types.PairReward]
}

func (p PairRewardsIndexes) IndexerList() []collections.Indexer[uint64, types.PairReward] {
	return []collections.Indexer[uint64, types.PairReward]{p.RewardsByPair}
}

// NewKeeper constructs a new keeper for oracle
func NewKeeper(cdc codec.BinaryCodec, storeKey sdk.StoreKey,
	paramspace paramstypes.Subspace, accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper, distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper, distrName string) Keeper {
	// ensure oracle module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	// set KeyTable if it has not already been set
	if !paramspace.HasKeyTable() {
		paramspace = paramspace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		paramSpace:        paramspace,
		AccountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		distrKeeper:       distrKeeper,
		StakingKeeper:     stakingKeeper,
		distrModuleName:   distrName,
		ExchangeRates:     collections.NewMap(storeKey, 1, asset.PairKeyEncoder, collections.DecValueEncoder),
		PriceSnapshots:    collections.NewMap(storeKey, 10, collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder), collections.ProtoValueEncoder[types.PriceSnapshot](cdc)),
		FeederDelegations: collections.NewMap(storeKey, 2, collections.ValAddressKeyEncoder, collections.AccAddressValueEncoder),
		MissCounters:      collections.NewMap(storeKey, 3, collections.ValAddressKeyEncoder, collections.Uint64ValueEncoder),
		Prevotes:          collections.NewMap(storeKey, 4, collections.ValAddressKeyEncoder, collections.ProtoValueEncoder[types.AggregateExchangeRatePrevote](cdc)),
		Votes:             collections.NewMap(storeKey, 5, collections.ValAddressKeyEncoder, collections.ProtoValueEncoder[types.AggregateExchangeRateVote](cdc)),
		WhitelistedPairs:  collections.NewKeySet(storeKey, 6, asset.PairKeyEncoder),
		PairRewards: collections.NewMap(
			storeKey, 7,
			collections.Uint64KeyEncoder, collections.ProtoValueEncoder[types.PairReward](cdc)),
		PairRewardsID: collections.NewSequence(storeKey, 9),
	}
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

/*
calcTwap walks through a slice of PriceSnapshots and tallies up the prices weighted
by the amount of time they were active for.

NOTE: Callers of this function should check if the snapshot slice is empty before
calling 'calcTwap'.
*/
func (k Keeper) calcTwap(ctx sdk.Context, snapshots []types.PriceSnapshot) (price sdk.Dec, err error) {
	if len(snapshots) == 0 {
		return price, errors.New("cannot calculate TWAP with empty snapshot slice")
	} else if len(snapshots) == 1 {
		return snapshots[0].Price, nil
	}
	twapLookBack := k.GetParams(ctx).TwapLookbackWindow.Milliseconds()
	firstTimeStamp := ctx.BlockTime().UnixMilli() - twapLookBack
	cumulativePrice := sdk.ZeroDec()

	firstTimeStamp = math.MaxInt64(snapshots[0].TimestampMs, firstTimeStamp)

	for i, s := range snapshots {
		var nextTimestampMs int64
		var timestampStart int64

		if i == 0 {
			timestampStart = firstTimeStamp
		} else {
			timestampStart = s.TimestampMs
		}

		if i == len(snapshots)-1 {
			// if we're at the last snapshot, then consider that price as ongoing until the current blocktime
			nextTimestampMs = ctx.BlockTime().UnixMilli()
		} else {
			nextTimestampMs = snapshots[i+1].TimestampMs
		}

		price := s.Price.MulInt64(nextTimestampMs - timestampStart)
		cumulativePrice = cumulativePrice.Add(price)
	}
	return cumulativePrice.QuoInt64(ctx.BlockTime().UnixMilli() - firstTimeStamp), nil
}

func (k Keeper) GetExchangeRateTwap(ctx sdk.Context, pair asset.Pair) (price sdk.Dec, err error) {
	snapshots := k.PriceSnapshots.Iterate(
		ctx,
		collections.PairRange[asset.Pair, time.Time]{}.
			Prefix(pair).
			StartInclusive(
				ctx.BlockTime().Add(-1*k.GetParams(ctx).TwapLookbackWindow)).
			EndInclusive(
				ctx.BlockTime()),
	).Values()

	if len(snapshots) == 0 {
		// if there are no snapshots, return -1 for the price
		return sdk.OneDec().Neg(), types.ErrNoValidTWAP
	}

	return k.calcTwap(ctx, snapshots)
}

func (k Keeper) GetExchangeRate(ctx sdk.Context, pair asset.Pair) (price sdk.Dec, err error) {
	return k.ExchangeRates.Get(ctx, pair)
}

// SetPrice sets the price for a pair as well as the price snapshot.
func (k Keeper) SetPrice(ctx sdk.Context, pair asset.Pair, price sdk.Dec) {
	k.ExchangeRates.Insert(ctx, pair, price)

	key := collections.Join(pair, ctx.BlockTime())
	timestampMs := ctx.BlockTime().UnixMilli()
	k.PriceSnapshots.Insert(ctx, key, types.PriceSnapshot{
		Pair:        pair,
		Price:       price,
		TimestampMs: timestampMs,
	})
	if err := ctx.EventManager().EmitTypedEvent(&types.OraclePriceUpdate{
		Pair:        pair.String(),
		Price:       price,
		TimestampMs: timestampMs,
	}); err != nil {
		ctx.Logger().Error("failed to emit OraclePriceUpdate", "pair", pair, "error", err)
	}
}
