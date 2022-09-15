package keeper

import (
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

/*
GetSpotPrice retrieves the price of the base asset denominated in quote asset.

The convention is the amount of quote assets required to buy one base asset.

e.g. If the tokenPair is BTC:NUSD, the method would return sdk.Dec(40,000.00)
because the instantaneous tangent slope on the vpool curve is 40,000.00,
so it would cost ~40,000.00 to buy one BTC:NUSD perp.

args:
  - ctx: cosmos-sdk context
  - pair: the token pair to get price for

ret:
  - price: the price of the token pair as sdk.Dec
  - err: error
*/
func (k Keeper) GetSpotPrice(ctx sdk.Context, pair common.AssetPair) (sdk.Dec, error) {
	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	if pool.BaseAssetReserve.IsNil() || pool.BaseAssetReserve.IsZero() ||
		pool.QuoteAssetReserve.IsNil() || pool.QuoteAssetReserve.IsZero() {
		return sdk.ZeroDec(), nil
	}

	return pool.QuoteAssetReserve.Quo(pool.BaseAssetReserve), nil
}

/*
So how much stablecoin you would get if you sold baseAssetAmount amount of perpetual contracts.

Returns the amount of quote assets required to achieve a move of baseAssetAmount in a direction.
e.g. if removing <baseAssetAmount> base assets from the pool, returns the amount of quote assets do so.

args:
  - ctx: cosmos-sdk context
  - pair: the trading token pair
  - dir: add or remove
  - baseAssetAmount: the amount of base asset

ret:
  - quoteAmount: the amount of quote assets required to make the desired swap
  - err: error
*/
func (k Keeper) GetBaseAssetPrice(
	ctx sdk.Context,
	pair common.AssetPair,
	dir types.Direction,
	baseAssetAmount sdk.Dec,
) (quoteAmount sdk.Dec, err error) {
	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	return pool.GetQuoteAmountByBaseAmount(dir, baseAssetAmount)
}

/*
Returns the amount of base assets required to achieve a move of quoteAmount in a direction.
e.g. if removing <quoteAmount> quote assets from the pool, returns the amount of base assets do so.

args:
  - ctx: cosmos-sdk context
  - pair: the trading token pair
  - dir: add or remove
  - quoteAmount: the amount of quote asset

ret:
  - baseAssetAmount: the amount of base assets required to make the desired swap
  - err: error
*/
func (k Keeper) GetQuoteAssetPrice(
	ctx sdk.Context,
	pair common.AssetPair,
	dir types.Direction,
	quoteAmount sdk.Dec,
) (baseAssetAmount sdk.Dec, err error) {
	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	return pool.GetBaseAmountByQuoteAmount(dir, quoteAmount)
}

/*
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
	pair common.AssetPair,
	direction types.Direction,
	baseAssetAmount sdk.Dec,
	lookbackInterval time.Duration,
) (quoteAssetAmount sdk.Dec, err error) {
	return k.calcTwap(
		ctx,
		pair,
		types.TwapCalcOption_BASE_ASSET_SWAP,
		direction,
		baseAssetAmount,
		lookbackInterval,
	)
}

/*
Returns the amount of base assets required to achieve a move of quoteAssetAmount in a direction,
based on historical snapshots.
e.g. if removing <quoteAssetAmount> quote assets from the pool, returns the amount of base assets do so.

args:
  - ctx: cosmos-sdk context
  - pair: the token pair
  - direction: add or remove
  - quoteAssetAmount: amount of base asset to add or remove
  - lookbackInterval: how far back to calculate TWAP

ret:
  - baseAssetAmount: the amount of quote asset to make the desired move, as sdk.Dec
  - err: error
*/
func (k Keeper) GetQuoteAssetTWAP(
	ctx sdk.Context,
	pair common.AssetPair,
	direction types.Direction,
	quoteAssetAmount sdk.Dec,
	lookbackInterval time.Duration,
) (baseAssetAmount sdk.Dec, err error) {
	return k.calcTwap(
		ctx,
		pair,
		types.TwapCalcOption_QUOTE_ASSET_SWAP,
		direction,
		quoteAssetAmount,
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
	pair common.AssetPair,
	twapCalcOption types.TwapCalcOption,
	direction types.Direction,
	assetAmount sdk.Dec,
	lookbackInterval time.Duration,
) (price sdk.Dec, err error) {
	// earliest timestamp we'll look back until
	lowerLimitTimestampMs := ctx.BlockTime().Add(-lookbackInterval).UnixMilli()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SnapshotsKeyPrefix)
	iter := store.ReverseIterator(nil, nil)
	defer iter.Close()

	if !iter.Valid() {
		return sdk.OneDec().Neg(), types.ErrNoValidTWAP
	}

	var cumulativePrice sdk.Dec = sdk.ZeroDec()
	var cumulativePeriodMs int64 = 0
	var prevTimestampMs int64 = ctx.BlockTime().UnixMilli()
	var currentSnapshot types.ReserveSnapshot

	for ; iter.Valid(); iter.Next() {
		k.codec.MustUnmarshal(iter.Value(), &currentSnapshot)

		currentPrice, err := getPriceWithSnapshot(
			currentSnapshot,
			snapshotPriceOptions{
				pair:           pair,
				twapCalcOption: twapCalcOption,
				direction:      direction,
				assetAmount:    assetAmount,
			},
		)
		if err != nil {
			return sdk.Dec{}, err
		}

		var timeElapsedMs int64
		if currentSnapshot.TimestampMs <= lowerLimitTimestampMs {
			// current snapshot is below the lower limit
			timeElapsedMs = prevTimestampMs - lowerLimitTimestampMs
		} else {
			timeElapsedMs = prevTimestampMs - currentSnapshot.TimestampMs
		}
		cumulativePrice = cumulativePrice.Add(currentPrice.MulInt64(timeElapsedMs))
		cumulativePeriodMs += timeElapsedMs

		// end early if we're already beyond the lower limit timestamp
		if currentSnapshot.TimestampMs <= lowerLimitTimestampMs {
			break
		}

		prevTimestampMs = currentSnapshot.TimestampMs
	}

	// definition of TWAP
	return cumulativePrice.QuoInt64(cumulativePeriodMs), nil
}
