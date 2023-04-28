package keeper

import (
	"time"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/asset"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
MarkPriceTWAP
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
func (k Keeper) MarkPriceTWAP(
	ctx sdk.Context,
	pair asset.Pair,
	lookbackInterval time.Duration,
) (quoteAssetAmount sdk.Dec, err error) {
	return k.CalcTwap(
		ctx,
		pair,
		v2types.TwapCalcOption_SPOT,
		v2types.Direction_DIRECTION_UNSPECIFIED, // unused
		sdk.ZeroDec(),                           // unused
		lookbackInterval,
	)
}

/*
BaseAssetTWAP
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
func (k Keeper) BaseAssetTWAP(
	ctx sdk.Context,
	pair asset.Pair,
	direction v2types.Direction,
	baseAssetAmount sdk.Dec,
	lookbackInterval time.Duration,
) (quoteAssetAmount sdk.Dec, err error) {
	return k.CalcTwap(
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
func (k Keeper) CalcTwap(
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
