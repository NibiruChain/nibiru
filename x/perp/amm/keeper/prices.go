package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

/*
GetMarkPrice retrieves the price of the base asset denominated in quote asset.

The convention is the amount of quote assets required to buy one base asset.

e.g. If the tokenPair is BTC:NUSD, the method would return sdk.Dec(40,000.00)
because the instantaneous tangent slope on the market curve is 40,000.00,
so it would cost ~40,000.00 to buy one BTC:NUSD perp.

args:
  - ctx: cosmos-sdk context
  - pair: the token pair to get price for

ret:
  - price: the price of the token pair as sdk.dec
  - err: error
*/
func (k Keeper) GetMarkPrice(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error) {
	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	return pool.GetMarkPrice(), nil
}

/*
GetBaseAssetPrice
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
	amm types.Market,
	dir types.Direction,
	baseAssetAmount sdk.Dec,
) (quoteAsset sdk.Dec, err error) {
	quoteReserve, err := amm.GetQuoteReserveByBase(baseAssetAmount.MulInt64(dir.ToMultiplier()))
	if err != nil {
		return
	}
	quoteAsset = amm.FromQuoteReserveToAsset(quoteReserve)
	return
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
		types.TwapCalcOption_SPOT,
		types.Direction_DIRECTION_UNSPECIFIED, // unused
		sdk.ZeroDec(),                         // unused
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
	twapCalcOption types.TwapCalcOption,
	direction types.Direction,
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

	var snapshots []types.ReserveSnapshot
	for ; iter.Valid(); iter.Next() {
		s := iter.Value()
		snapshots = append(snapshots, s)
		if s.TimestampMs <= lowerLimitTimestampMs {
			break
		}
	}

	if len(snapshots) == 0 {
		return sdk.OneDec().Neg(), types.ErrNoValidTWAP
	}

	fmt.Println(snapshots)

	return calcTwap(ctx, snapshots, lowerLimitTimestampMs, twapCalcOption, direction, assetAmount)
}

// calcTwap walks through a slice of PriceSnapshots and tallies up the prices weighted by the amount of time they were active for.
// Callers of this function should already check if the snapshot slice is empty. Passing an empty snapshot slice will result in a panic.
func calcTwap(ctx sdk.Context, snapshots []types.ReserveSnapshot, lowerLimitTimestampMs int64, twapCalcOption types.TwapCalcOption, direction types.Direction, assetAmt sdk.Dec) (sdk.Dec, error) {
	// circuit-breaker when there's only one snapshot to process
	if len(snapshots) == 1 {
		return getPriceWithSnapshot(
			snapshots[0],
			snapshotPriceOptions{
				pair:           snapshots[0].Pair,
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
				pair:           s.Pair,
				twapCalcOption: twapCalcOption,
				direction:      direction,
				assetAmount:    assetAmt,
			},
		)
		fmt.Println(s.String())
		fmt.Println(sPrice)
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
