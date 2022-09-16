package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func NewKeeper(
	codec codec.BinaryCodec,
	storeKey sdk.StoreKey,
	pricefeedKeeper types.PricefeedKeeper,
) Keeper {
	return Keeper{
		codec:           codec,
		storeKey:        storeKey,
		pricefeedKeeper: pricefeedKeeper,
	}
}

type Keeper struct {
	codec           codec.BinaryCodec
	storeKey        sdk.StoreKey
	pricefeedKeeper types.PricefeedKeeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

/*
Trades baseAssets in exchange for quoteAssets.
The base asset is a crypto asset like BTC.
The quote asset is a stablecoin like NUSD.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - baseAssetAmount: the amount of quote asset being traded
  - quoteAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - quoteAssetAmount: the amount of quote asset swapped
  - err: error
*/
func (k Keeper) SwapBaseForQuote(
	ctx sdk.Context,
	pair common.AssetPair,
	dir types.Direction,
	baseAmt sdk.Dec,
	quoteLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (quoteAmt sdk.Dec, err error) {
	if !k.ExistsPool(ctx, pair) {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	if !k.pricefeedKeeper.IsActivePair(ctx, pair.String()) {
		return sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", pair.String())
	}

	if baseAmt.IsZero() {
		return sdk.ZeroDec(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !pool.HasEnoughBaseReserve(baseAmt) {
		return sdk.Dec{}, types.ErrOverTradingLimit
	}

	quoteAmt, err = pool.GetQuoteAmountByBaseAmount(dir, baseAmt)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !pool.HasEnoughQuoteReserve(quoteAmt) {
		// in reality, this if statement should never run because a perturbation in quote reserve assets
		// greater than trading limit ratio would have happened when checking for a perturbation in
		// base assets, due to x*y=k
		//
		// e.g. a 10% change in quote asset reserves would have triggered a >10% change in
		// base asset reserves
		return sdk.Dec{}, types.ErrOverTradingLimit.Wrapf(
			"quote amount %s is over trading limit", quoteAmt)
	}

	if !quoteLimit.IsZero() {
		// if going long and the base amount retrieved from the pool is less than the limit
		if dir == types.Direction_ADD_TO_POOL && quoteAmt.LT(quoteLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"quote amount (%s) is less than selected limit (%s)",
				quoteAmt.String(),
				quoteLimit.String(),
			)
			// if going short and the base amount retrieved from the pool is greater than the limit
		} else if dir == types.Direction_REMOVE_FROM_POOL && quoteAmt.GT(quoteLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"quote amount (%s) is greater than selected limit (%s)",
				quoteAmt.String(),
				quoteLimit.String(),
			)
		}
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.IncreaseBaseAssetReserve(baseAmt)
		pool.DecreaseQuoteAssetReserve(quoteAmt)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.DecreaseBaseAssetReserve(baseAmt)
		pool.IncreaseQuoteAssetReserve(quoteAmt)
	}

	if err = k.updatePool(ctx, pool, skipFluctuationLimitCheck); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	spotPrice, err := k.GetSpotPrice(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.MarkPriceChangedEvent{
		Pair:      pair.String(),
		Price:     spotPrice,
		Timestamp: ctx.BlockHeader().Time,
	}); err != nil {
		return sdk.Dec{}, err
	}

	return quoteAmt, ctx.EventManager().EmitTypedEvent(&types.SwapBaseForQuoteEvent{
		Pair:        pair.String(),
		QuoteAmount: quoteAmt,
		BaseAmount:  baseAmt,
	})
}

/*
Trades quoteAssets in exchange for baseAssets.
The quote asset is a stablecoin like NUSD.
The base asset is a crypto asset like BTC or ETH.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - quoteAssetAmount: the amount of quote asset being traded
  - baseAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - baseAssetAmount: the amount of base asset swapped
  - err: error
*/
func (k Keeper) SwapQuoteForBase(
	ctx sdk.Context,
	pair common.AssetPair,
	dir types.Direction,
	quoteAmt sdk.Dec,
	baseLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (baseAmt sdk.Dec, err error) {
	if !k.ExistsPool(ctx, pair) {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	if !k.pricefeedKeeper.IsActivePair(ctx, pair.String()) {
		return sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", pair.String())
	}

	if quoteAmt.IsZero() {
		return sdk.ZeroDec(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	// check trade limit ratio on quote in either direction
	if !pool.HasEnoughQuoteReserve(quoteAmt) {
		return sdk.Dec{}, types.ErrOverTradingLimit.Wrapf(
			"quote amount %s is over trading limit", quoteAmt)
	}

	baseAmt, err = pool.GetBaseAmountByQuoteAmount(dir, quoteAmt)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !pool.HasEnoughBaseReserve(baseAmt) {
		// in reality, this if statement should never run because a perturbation in base reserve assets
		// greater than trading limit ratio would have happened when checking for a perturbation in
		// quote assets, due to x*y=k
		//
		// e.g. a 10% change in base asset reserves would have triggered a >10% change in
		// quote asset reserves
		return sdk.Dec{}, types.ErrOverTradingLimit.Wrapf(
			"base amount %s is over trading limit", baseAmt)
	}

	// check if base asset limit is violated
	if !baseLimit.IsZero() {
		// if going long and the base amount retrieved from the pool is less than the limit
		if dir == types.Direction_ADD_TO_POOL && baseAmt.LT(baseLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"base amount (%s) is less than selected limit (%s)",
				baseAmt.String(),
				baseLimit.String(),
			)
			// if going short and the base amount retrieved from the pool is greater than the limit
		} else if dir == types.Direction_REMOVE_FROM_POOL && baseAmt.GT(baseLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"base amount (%s) is greater than selected limit (%s)",
				baseAmt.String(),
				baseLimit.String(),
			)
		}
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.DecreaseBaseAssetReserve(baseAmt)
		pool.IncreaseQuoteAssetReserve(quoteAmt)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.IncreaseBaseAssetReserve(baseAmt)
		pool.DecreaseQuoteAssetReserve(quoteAmt)
	}

	if err = k.updatePool(ctx, pool, skipFluctuationLimitCheck); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	spotPrice, err := k.GetSpotPrice(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.MarkPriceChangedEvent{
		Pair:      pair.String(),
		Price:     spotPrice,
		Timestamp: ctx.BlockHeader().Time,
	}); err != nil {
		return sdk.Dec{}, err
	}

	return baseAmt, ctx.EventManager().EmitTypedEvent(&types.SwapQuoteForBaseEvent{
		Pair:        pair.String(),
		QuoteAmount: quoteAmt,
		BaseAmount:  baseAmt,
	})
}

/*
*
Check's that a pool that we're about to save to state does not violate the fluctuation limit.
Always tries to check against a snapshot from a previous block. If one doesn't exist, then it just uses the current snapshot.
This should run prior to updating the snapshot, otherwise it will compare the currently updated vpool to itself.

args:
  - ctx: the cosmos-sdk context
  - pool: the updated vpool

ret:
  - err: error if any
*/
func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool *types.VPool) error {
	if pool.FluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	latestSnapshot, err := k.GetLatestReserveSnapshot(ctx, pool.Pair)
	if err != nil {
		return fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
	}

	if isOverFluctuationLimit(pool, latestSnapshot) {
		return types.ErrOverFluctuationLimit
	}

	return nil
}

/*
*
isOverFluctuationLimit compares the updated pool's spot price with the current spot price.

If the fluctuation limit ratio is zero, then the fluctuation limit check is skipped.

args:
  - pool: the updated vpool
  - snapshot: the snapshot to compare against

ret:
  - bool: true if the fluctuation limit is violated. false otherwise
*/
func isOverFluctuationLimit(pool *types.VPool, snapshot types.ReserveSnapshot) bool {
	if pool.FluctuationLimitRatio.IsZero() {
		return false
	}

	price := pool.QuoteAssetReserve.Quo(pool.BaseAssetReserve)

	lastPrice := snapshot.QuoteAssetReserve.Quo(snapshot.BaseAssetReserve)
	upperLimit := lastPrice.Mul(sdk.OneDec().Add(pool.FluctuationLimitRatio))
	lowerLimit := lastPrice.Mul(sdk.OneDec().Sub(pool.FluctuationLimitRatio))

	if price.GT(upperLimit) || price.LT(lowerLimit) {
		return true
	}

	return false
}

/*
*
IsOverSpreadLimit compares the current spot price of the vpool (given by pair) to the underlying's index price (given by an oracle).
It panics if you provide it with a pair that doesn't exist in the state.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - bool: whether or not the price has deviated from the oracle price beyond a spread ratio
*/
func (k Keeper) IsOverSpreadLimit(ctx sdk.Context, pair common.AssetPair) bool {
	spotPrice, err := k.GetSpotPrice(ctx, pair)
	if err != nil {
		panic(err)
	}

	indexPrice, err := k.pricefeedKeeper.GetCurrentPrice(ctx, pair.BaseDenom(), pair.QuoteDenom())
	if err != nil {
		panic(err)
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		panic(err)
	}

	return spotPrice.Sub(indexPrice.Price).Quo(indexPrice.Price).Abs().GTE(pool.MaxOracleSpreadRatio)
}

/*
*
GetMaintenanceMarginRatio returns the maintenance margin ratio for the pool from the asset pair.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - sdk.Dec: The maintenance margin ratio for the pool
*/
func (k Keeper) GetMaintenanceMarginRatio(ctx sdk.Context, pair common.AssetPair) sdk.Dec {
	pool, err := k.getPool(ctx, pair)
	if err != nil {
		panic(err)
	}

	return pool.MaintenanceMarginRatio
}

/*
*
GetMaxLeverage returns the maximum leverage required to open a position in the pool.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - sdk.Dec: The maintenance margin ratio for the pool
*/
func (k Keeper) GetMaxLeverage(ctx sdk.Context, pair common.AssetPair) sdk.Dec {
	pool, err := k.getPool(ctx, pair)
	if err != nil {
		panic(err)
	}

	return pool.MaxLeverage
}
