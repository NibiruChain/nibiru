package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      sdk.StoreKey
	ParamSubspace paramtypes.Subspace

	BankKeeper    types.BankKeeper
	AccountKeeper types.AccountKeeper
	OracleKeeper  types.OracleKeeper
	EpochKeeper   types.EpochKeeper

	Markets          collections.Map[asset.Pair, v2types.Market]
	AMMs             collections.Map[asset.Pair, v2types.AMM]
	Positions        collections.Map[collections.Pair[asset.Pair, sdk.AccAddress], v2types.Position]
	Metrics          collections.Map[asset.Pair, v2types.Metrics]
	ReserveSnapshots collections.Map[collections.Pair[asset.Pair, time.Time], v2types.ReserveSnapshot]
}

// NewKeeper Creates a new x/perp Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	paramSubspace paramtypes.Subspace,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
	epochKeeper types.EpochKeeper,
) Keeper {
	// Ensure that the module account is set.
	if moduleAcc := accountKeeper.GetModuleAddress(types.ModuleName); moduleAcc == nil {
		panic("The x/perp module account has not been set")
	}

	// Set param.types.'KeyTable' if it has not already been set
	if !paramSubspace.HasKeyTable() {
		paramSubspace = paramSubspace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		ParamSubspace: paramSubspace,
		BankKeeper:    bankKeeper,
		AccountKeeper: accountKeeper,
		OracleKeeper:  oracleKeeper,
		EpochKeeper:   epochKeeper,
		Markets: collections.NewMap(
			storeKey, 0,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[v2types.Market](cdc),
		),
		AMMs: collections.NewMap(
			storeKey, 1,
			asset.PairKeyEncoder,
			collections.ProtoValueEncoder[v2types.AMM](cdc),
		),
		Positions: collections.NewMap(
			storeKey, 2,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.AccAddressKeyEncoder),
			collections.ProtoValueEncoder[v2types.Position](cdc),
		),
		Metrics: collections.NewMap(storeKey, 3, asset.PairKeyEncoder, collections.ProtoValueEncoder[v2types.Metrics](cdc)),
		ReserveSnapshots: collections.NewMap(
			storeKey, 4,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder),
			collections.ProtoValueEncoder[v2types.ReserveSnapshot](cdc),
		),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", v2types.ModuleName))
}

// GetParams get all parameters as v2types.Params
func (k Keeper) GetParams(ctx sdk.Context) (params v2types.Params) {
	k.ParamSubspace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params v2types.Params) {
	k.ParamSubspace.SetParamSet(ctx, &params)
}

/*
GetMarkPriceTWAP
Returns the twap of the spot price (y/x).

args:
  - ctx: cosmos-sdk context
  - pair: the token pair
  - direction: add or remove
  - baseAssetAmount: amount of base asset to add or remove
  - lookbackInterval: how far back to calculate TWAP

ret:
  - quoteAssetAmount: the amount of quote asset to make the desired move, as sdk.Dec
  - err: error
*/
func (k Keeper) GetMarkPriceTWAP(
	ctx sdk.Context,
	pair asset.Pair,
	lookbackInterval time.Duration,
) (quoteAssetAmount sdk.Dec, err error) {
	return k.calcTwap(
		ctx,
		pair,
		v2types.TwapCalcOption_SPOT,
		v2types.Direction_DIRECTION_UNSPECIFIED, // unused
		sdk.ZeroDec(),                           // unused
		lookbackInterval,
	)
}

/*
GetBaseAssetTWAP
Returns the amount of quote assets required to achieve a move of baseAssetAmount in a direction,
based on historical snapshots.
e.g. if removing <baseAssetAmount> base assets from the pool, returns the amount of quote assets do so.

args:
  - ctx: cosmos-sdk context
  - pair: the token pair
  - direction: add or remove
  - baseAssetAmount: amount of base asset to add or remove
  - lookbackInterval: how far back to calculate TWAP

ret:
  - quoteAssetAmount: the amount of quote asset to make the desired move, as sdk.Dec
  - err: error
*/
func (k Keeper) GetBaseAssetTWAP(
	ctx sdk.Context,
	pair asset.Pair,
	direction v2types.Direction,
	baseAssetAmount sdk.Dec,
	lookbackInterval time.Duration,
) (quoteAssetAmount sdk.Dec, err error) {
	return k.calcTwap(
		ctx,
		pair,
		v2types.TwapCalcOption_BASE_ASSET_SWAP,
		direction,
		baseAssetAmount,
		lookbackInterval,
	)
}

/*
Gets the time-weighted average price from [ ctx.BlockTime() - interval, ctx.BlockTime() )
Note the open-ended right bracket.

args:
  - ctx: cosmos-sdk context
  - pair: the token pair
  - twapCalcOption: one of SPOT, QUOTE_ASSET_SWAP, or BASE_ASSET_SWAP
  - direction: add or remove, only required for QUOTE_ASSET_SWAP or BASE_ASSET_SWAP
  - assetAmount: amount of asset to add or remove, only required for QUOTE_ASSET_SWAP or BASE_ASSET_SWAP
  - lookbackInterval: how far back to calculate TWAP

ret:
  - price: TWAP as sdk.Dec
  - err: error
*/
func (k Keeper) calcTwap(
	ctx sdk.Context,
	pair asset.Pair,
	twapCalcOption v2types.TwapCalcOption,
	direction v2types.Direction,
	assetAmount sdk.Dec,
	lookbackInterval time.Duration,
) (price sdk.Dec, err error) {
	// earliest timestamp we'll look back until
	lowerLimitTimestampMs := ctx.BlockTime().Add(-1 * lookbackInterval).UnixMilli()

	iter := k.ReserveSnapshots.Iterate(
		ctx,
		collections.PairRange[asset.Pair, time.Time]{}.
			Prefix(pair).
			EndInclusive(ctx.BlockTime()).
			Descending(),
	)
	defer iter.Close()

	var snapshots []v2types.ReserveSnapshot
	for ; iter.Valid(); iter.Next() {
		s := iter.Value()
		snapshots = append(snapshots, s)
		if s.TimestampMs <= lowerLimitTimestampMs {
			break
		}
	}

	if len(snapshots) == 0 {
		return sdk.OneDec().Neg(), v2types.ErrNoValidTWAP
	}

	return calcTwap(ctx, snapshots, lowerLimitTimestampMs, twapCalcOption, direction, assetAmount)
}

// calcTwap walks through a slice of PriceSnapshots and tallies up the prices weighted by the amount of time they were active for.
// Callers of this function should already check if the snapshot slice is empty. Passing an empty snapshot slice will result in a panic.
func calcTwap(ctx sdk.Context, snapshots []v2types.ReserveSnapshot, lowerLimitTimestampMs int64, twapCalcOption v2types.TwapCalcOption, direction v2types.Direction, assetAmt sdk.Dec) (sdk.Dec, error) {
	// circuit-breaker when there's only one snapshot to process
	if len(snapshots) == 1 {
		return getPriceWithSnapshot(
			snapshots[0],
			snapshotPriceOptions{
				pair:           snapshots[0].Amm.Pair,
				twapCalcOption: twapCalcOption,
				direction:      direction,
				assetAmount:    assetAmt,
			},
		)
	}

	prevTimestampMs := ctx.BlockTime().UnixMilli()
	cumulativePrice := sdk.ZeroDec()
	cumulativePeriodMs := int64(0)

	for _, s := range snapshots {
		sPrice, err := getPriceWithSnapshot(
			s,
			snapshotPriceOptions{
				pair:           s.Amm.Pair,
				twapCalcOption: twapCalcOption,
				direction:      direction,
				assetAmount:    assetAmt,
			},
		)
		if err != nil {
			return sdk.Dec{}, err
		}
		var timeElapsedMs int64
		if s.TimestampMs <= lowerLimitTimestampMs {
			// if we're at a snapshot below lowerLimitTimestamp, then consider that price as starting from the lower limit timestamp
			timeElapsedMs = prevTimestampMs - lowerLimitTimestampMs
		} else {
			timeElapsedMs = prevTimestampMs - s.TimestampMs
		}
		cumulativePrice = cumulativePrice.Add(sPrice.MulInt64(timeElapsedMs))
		cumulativePeriodMs += timeElapsedMs
		if s.TimestampMs <= lowerLimitTimestampMs {
			break
		}
		prevTimestampMs = s.TimestampMs
	}
	twap := cumulativePrice.QuoInt64(cumulativePeriodMs)
	return twap, nil
}

/*
An object parameter for getPriceWithSnapshot().

Specifies how to read the price from a single snapshot. There are three ways:
SPOT: spot price
QUOTE_ASSET_SWAP: price when swapping y amount of quote assets
BASE_ASSET_SWAP: price when swapping x amount of base assets
*/
type snapshotPriceOptions struct {
	// required
	pair           asset.Pair
	twapCalcOption v2types.TwapCalcOption

	// required only if twapCalcOption == QUOTE_ASSET_SWAP or BASE_ASSET_SWAP
	direction   v2types.Direction
	assetAmount sdk.Dec
}

/*
Pure function that returns a price from a snapshot.

Can choose from three types of calc options: SPOT, QUOTE_ASSET_SWAP, and BASE_ASSET_SWAP.
QUOTE_ASSET_SWAP and BASE_ASSET_SWAP require the `direction“ and `assetAmount“ args.
SPOT does not require `direction` and `assetAmount`.

args:
  - pair: the token pair
  - snapshot: a reserve snapshot
  - twapCalcOption: SPOT, QUOTE_ASSET_SWAP, or BASE_ASSET_SWAP
  - direction: add or remove; only required for QUOTE_ASSET_SWAP or BASE_ASSET_SWAP
  - assetAmount: the amount of base or quote asset; only required for QUOTE_ASSET_SWAP or BASE_ASSET_SWAP

ret:
  - price: the price as sdk.Dec
  - err: error
*/
func getPriceWithSnapshot(
	snapshot v2types.ReserveSnapshot,
	snapshotPriceOpts snapshotPriceOptions,
) (price sdk.Dec, err error) {
	switch snapshotPriceOpts.twapCalcOption {
	case v2types.TwapCalcOption_SPOT:
		return snapshot.Amm.QuoteReserve.Mul(snapshot.Amm.PriceMultiplier).Quo(snapshot.Amm.BaseReserve), nil

	case v2types.TwapCalcOption_QUOTE_ASSET_SWAP:
		quoteReserve := snapshot.Amm.FromQuoteAssetToReserve(snapshotPriceOpts.assetAmount)
		return snapshot.Amm.GetBaseReserveAmt(quoteReserve, snapshotPriceOpts.direction)

	case v2types.TwapCalcOption_BASE_ASSET_SWAP:
		quoteReserve, err := snapshot.Amm.GetQuoteReserveAmt(snapshotPriceOpts.assetAmount, snapshotPriceOpts.direction)
		if err != nil {
			return sdk.ZeroDec(), err
		}
		return snapshot.Amm.FromQuoteReserveToAsset(quoteReserve), nil
	}

	return sdk.ZeroDec(), nil
}

// checkUserLimits checks if the limit is violated by the amount.
// returns error if it does
func checkUserLimits(limit, amount sdk.Dec, dir v2types.Direction) error {
	if limit.IsZero() {
		return nil
	}

	if dir == v2types.Direction_LONG && amount.LT(limit) {
		return v2types.ErrAssetFailsUserLimit.Wrapf(
			"amount (%s) is less than selected limit (%s)",
			amount.String(),
			limit.String(),
		)
	}

	if dir == v2types.Direction_SHORT && amount.GT(limit) {
		return v2types.ErrAssetFailsUserLimit.Wrapf(
			"amount (%s) is greater than selected limit (%s)",
			amount.String(),
			limit.String(),
		)
	}

	return nil
}

/*
*
Check's that a pool that we're about to save to state does not violate the fluctuation limit.
Always tries to check against a snapshot from a previous block. If one doesn't exist, then it just uses the current snapshot.
This should run prior to updating the snapshot, otherwise it will compare the currently updated market to itself.

args:
  - ctx: the cosmos-sdk context
  - pool: the updated market

ret:
  - err: error if any
*/
func (k Keeper) checkPriceFluctuationLimitRatio(ctx sdk.Context, market v2types.Market, amm v2types.AMM) error {
	if market.PriceFluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	lastSnapshot, err := k.GetLastSnapshot(ctx, market.Pair)
	if err != nil {
		return err
	}
	if market.IsOverFluctuationLimitInRelationWithSnapshot(amm, lastSnapshot) {
		return v2types.ErrOverFluctuationLimit
	}

	return nil
}

/*
GetLastSnapshot retrieve the last snapshot for a particular market

args:
  - ctx: the cosmos-sdk context
  - pool: the market to check
*/
func (k Keeper) GetLastSnapshot(ctx sdk.Context, pair asset.Pair) (v2types.ReserveSnapshot, error) {
	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}.Prefix(pair).Descending())
	defer it.Close()

	if !it.Valid() {
		return v2types.ReserveSnapshot{}, fmt.Errorf("error getting last snapshot number for pair %s", pair)
	}

	return it.Value(), nil
}

/*
IsOverSpreadLimit compares the current spot price of the market (given by pair) to the underlying's index price (given by an oracle).
It panics if you provide it with a pair that doesn't exist in the state.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - bool: whether the price has deviated from the oracle price beyond a spread ratio
*/
func (k Keeper) IsOverSpreadLimit(ctx sdk.Context, market v2types.Market, amm v2types.AMM) (bool, error) {
	indexPrice, err := k.OracleKeeper.GetExchangeRate(ctx, amm.Pair)
	if err != nil {
		return false, err
	}

	priceDeltaAbs := amm.MarkPrice().Sub(indexPrice).Abs()

	return priceDeltaAbs.Quo(indexPrice).GTE(market.MaxOracleSpreadRatio), nil
}
