package keeper

import (
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

/*
CalcTwap Gets the time-weighted average price from [ ctx.BlockTime() - interval, ctx.BlockTime() )
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
	twapCalcOption types.TwapCalcOption,
	direction types.Direction,
	assetAmt sdk.Dec,
	lookbackInterval time.Duration,
) (price sdk.Dec, err error) {
	// earliest timestamp we'll look back until
	lowerLimitTimestampMs := ctx.BlockTime().Add(-1 * lookbackInterval).UnixMilli()

	// fetch snapshots from state
	var snapshots []types.ReserveSnapshot
	iter := k.ReserveSnapshots.Iterate(
		ctx,
		collections.PairRange[asset.Pair, time.Time]{}.
			Prefix(pair).
			EndInclusive(ctx.BlockTime()).
			Descending(),
	)
	defer iter.Close()
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

	// circuit-breaker when there's only one snapshot to process
	if len(snapshots) == 1 {
		return getPriceWithSnapshot(
			snapshots[0],
			snapshotPriceOps{
				twapCalcOption: twapCalcOption,
				direction:      direction,
				assetAmt:       assetAmt,
			},
		)
	}

	// else, iterate over all snapshots and calculate TWAP
	prevTimestampMs := ctx.BlockTime().UnixMilli()
	cumulativePrice := sdk.ZeroDec()
	cumulativePeriodMs := int64(0)

	for _, snapshot := range snapshots {
		if snapshot.TimestampMs == prevTimestampMs {
			// in some extreme cases, 2 reserves snapshots can have the same timestamp which would cause a divide by zero error
			// in this case, we skip the second snapshot
			continue
		}

		price, err := getPriceWithSnapshot(
			snapshot,
			snapshotPriceOps{
				twapCalcOption: twapCalcOption,
				direction:      direction,
				assetAmt:       assetAmt,
			},
		)
		if err != nil {
			return sdk.Dec{}, err
		}

		var timeElapsedMs int64
		if snapshot.TimestampMs <= lowerLimitTimestampMs {
			// if we're at a snapshot below lowerLimitTimestamp, then consider that price as starting from the lower limit timestamp
			timeElapsedMs = prevTimestampMs - lowerLimitTimestampMs
		} else {
			timeElapsedMs = prevTimestampMs - snapshot.TimestampMs
		}

		cumulativePrice = cumulativePrice.Add(price.MulInt64(timeElapsedMs))
		cumulativePeriodMs += timeElapsedMs

		if snapshot.TimestampMs <= lowerLimitTimestampMs {
			break
		}
		prevTimestampMs = snapshot.TimestampMs
	}

	if cumulativePeriodMs == 0 {
		// Should not be reachable
		return getPriceWithSnapshot(
			snapshots[0],
			snapshotPriceOps{
				twapCalcOption: twapCalcOption,
				direction:      direction,
				assetAmt:       assetAmt,
			},
		)
	}

	return cumulativePrice.QuoInt64(cumulativePeriodMs), nil
}

/*
An object parameter for getPriceWithSnapshot().

Specifies how to read the price from a single snapshot. There are three ways:
SPOT: spot price
QUOTE_ASSET_SWAP: price when swapping y amount of quote assets
BASE_ASSET_SWAP: price when swapping x amount of base assets
*/
type snapshotPriceOps struct {
	twapCalcOption types.TwapCalcOption
	direction      types.Direction
	assetAmt       sdk.Dec
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
	snapshot types.ReserveSnapshot,
	opts snapshotPriceOps,
) (price sdk.Dec, err error) {
	switch opts.twapCalcOption {
	case types.TwapCalcOption_SPOT:
		return snapshot.Amm.QuoteReserve.Mul(snapshot.Amm.PriceMultiplier).Quo(snapshot.Amm.BaseReserve), nil

	case types.TwapCalcOption_QUOTE_ASSET_SWAP:
		quoteReserve := snapshot.Amm.FromQuoteAssetToReserve(opts.assetAmt)
		return snapshot.Amm.GetBaseReserveAmt(quoteReserve, opts.direction)

	case types.TwapCalcOption_BASE_ASSET_SWAP:
		quoteReserve, err := snapshot.Amm.GetQuoteReserveAmt(opts.assetAmt, opts.direction)
		if err != nil {
			return sdk.Dec{}, err
		}
		return snapshot.Amm.FromQuoteReserveToAsset(quoteReserve), nil
	}

	return sdk.ZeroDec(), nil
}
