package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

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
	baseAssetAmount sdk.Dec,
	quoteAmountLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (quoteAssetAmount sdk.Dec, err error) {
	if !k.ExistsPool(ctx, pair) {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	if baseAssetAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == types.Direction_REMOVE_FROM_POOL && !pool.HasEnoughBaseReserve(baseAssetAmount) {
		return sdk.Dec{}, types.ErrOverTradingLimit
	}

	quoteAssetAmount, err = pool.GetQuoteAmountByBaseAmount(dir, baseAssetAmount)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !quoteAmountLimit.IsZero() {
		// if going long and the base amount retrieved from the pool is less than the limit
		if dir == types.Direction_ADD_TO_POOL && quoteAssetAmount.LT(quoteAmountLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"quote amount (%s) is less than selected limit (%s)",
				quoteAssetAmount.String(),
				quoteAmountLimit.String(),
			)
			// if going short and the base amount retrieved from the pool is greater than the limit
		} else if dir == types.Direction_REMOVE_FROM_POOL && quoteAssetAmount.GT(quoteAmountLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"quote amount (%s) is greater than selected limit (%s)",
				quoteAssetAmount.String(),
				quoteAmountLimit.String(),
			)
		}
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.IncreaseBaseAssetReserve(baseAssetAmount)
		pool.DecreaseQuoteAssetReserve(quoteAssetAmount)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.DecreaseBaseAssetReserve(baseAssetAmount)
		pool.IncreaseQuoteAssetReserve(quoteAssetAmount)
	}

	if err = k.updatePool(ctx, pool, skipFluctuationLimitCheck); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	spotPrice, err := k.GetSpotPrice(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.MarkPriceChanged{
		Pair:      pair.String(),
		Price:     spotPrice,
		Timestamp: ctx.BlockHeader().Time,
	}); err != nil {
		return sdk.Dec{}, err
	}

	return quoteAssetAmount, ctx.EventManager().EmitTypedEvent(&types.SwapBaseForQuoteEvent{
		Pair:        pair.String(),
		QuoteAmount: quoteAssetAmount,
		BaseAmount:  baseAssetAmount,
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
	quoteAssetAmount sdk.Dec,
	baseAmountLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (baseAssetAmount sdk.Dec, err error) {
	if !k.ExistsPool(ctx, pair) {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	if quoteAssetAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == types.Direction_REMOVE_FROM_POOL &&
		!pool.HasEnoughQuoteReserve(quoteAssetAmount) {
		return sdk.Dec{}, types.ErrOverTradingLimit
	}

	baseAssetAmount, err = pool.GetBaseAmountByQuoteAmount(dir, quoteAssetAmount)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !baseAmountLimit.IsZero() {
		// if going long and the base amount retrieved from the pool is less than the limit
		if dir == types.Direction_ADD_TO_POOL && baseAssetAmount.LT(baseAmountLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"base amount (%s) is less than selected limit (%s)",
				baseAssetAmount.String(),
				baseAmountLimit.String(),
			)
			// if going short and the base amount retrieved from the pool is greater than the limit
		} else if dir == types.Direction_REMOVE_FROM_POOL && baseAssetAmount.GT(baseAmountLimit) {
			return sdk.Dec{}, types.ErrAssetFailsUserLimit.Wrapf(
				"base amount (%s) is greater than selected limit (%s)",
				baseAssetAmount.String(),
				baseAmountLimit.String(),
			)
		}
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.DecreaseBaseAssetReserve(baseAssetAmount)
		pool.IncreaseQuoteAssetReserve(quoteAssetAmount)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.IncreaseBaseAssetReserve(baseAssetAmount)
		pool.DecreaseQuoteAssetReserve(quoteAssetAmount)
	}

	if err = k.updatePool(ctx, pool, skipFluctuationLimitCheck); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	spotPrice, err := k.GetSpotPrice(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}
	if err := ctx.EventManager().EmitTypedEvent(&types.MarkPriceChanged{
		Pair:      pair.String(),
		Price:     spotPrice,
		Timestamp: ctx.BlockHeader().Time,
	}); err != nil {
		return sdk.Dec{}, err
	}

	return baseAssetAmount, ctx.EventManager().EmitTypedEvent(&types.SwapQuoteForBaseEvent{
		Pair:        pair.String(),
		QuoteAmount: quoteAssetAmount,
		BaseAmount:  baseAssetAmount,
	})
}

/**
Check's that a pool that we're about to save to state does not violate the fluctuation limit.
Always tries to check against a snapshot from a previous block. If one doesn't exist, then it just uses the current snapshot.
This should run prior to updating the snapshot, otherwise it will compare the currently updated vpool to itself.

args:
  - ctx: the cosmos-sdk context
  - pool: the updated vpool

ret:
  - err: error if any
*/
func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool *types.Pool) error {
	if pool.FluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	latestSnapshot, counter, err := k.getLatestReserveSnapshot(ctx, pool.Pair)
	if err != nil {
		return fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
	}

	if latestSnapshot.BlockNumber == ctx.BlockHeight() && counter > 0 {
		latestSnapshot, err = k.getSnapshot(ctx, pool.Pair, counter-1)
		if err != nil {
			return fmt.Errorf("error getting snapshot number %d from pair %s", counter-1, pool.Pair)
		}
	}

	if isOverFluctuationLimit(pool, latestSnapshot) {
		return types.ErrOverFluctuationLimit
	}

	return nil
}

/**
isOverFluctuationLimit compares the updated pool's reserves with the given reserve snapshot, and errors if the fluctuation is above the bounds.

If the fluctuation limit ratio is zero, then the fluctuation limit check is skipped.

args:
  - pool: the updated vpool
  - snapshot: the snapshot to compare against

ret:
  - bool: true if the fluctuation limit is violated. false otherwise
*/
func isOverFluctuationLimit(pool *types.Pool, snapshot types.ReserveSnapshot) bool {
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

/**
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

	oraclePrice, err := k.GetUnderlyingPrice(ctx, pair)
	if err != nil {
		panic(err)
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		panic(err)
	}

	return spotPrice.Sub(oraclePrice).Quo(oraclePrice).Abs().GTE(pool.MaxOracleSpreadRatio)
}

/**
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
