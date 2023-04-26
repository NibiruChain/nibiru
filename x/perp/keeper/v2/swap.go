package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

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
  - skipFluctuationLimitCheck: whether or not to check if the swapped amount is over the fluctuation limit. Currently unused.

ret:
  - baseAssetAmount: the amount of base asset swapped
  - err: error
*/
func (k Keeper) SwapQuoteAsset(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	dir v2types.Direction,
	quoteAssetAmt sdk.Dec,
	baseAssetLimit sdk.Dec,
) (updatedAMM *v2types.AMM, baseAssetDelta sdk.Dec, err error) {
	if !quoteAssetAmt.IsPositive() {
		return &amm, sdk.ZeroDec(), nil
	}

	if _, err = k.OracleKeeper.GetExchangeRate(ctx, amm.Pair); err != nil {
		return nil, sdk.Dec{}, v2types.ErrNoValidPrice.Wrapf("%s", amm.Pair)
	}

	updatedAMM, baseAssetDelta, err = amm.SwapQuoteAsset(quoteAssetAmt, dir)
	if err != nil {
		return nil, sdk.Dec{}, err
	}

	if err := checkUserLimits(baseAssetLimit, baseAssetDelta, dir); err != nil {
		return nil, sdk.Dec{}, err
	}

	k.AMMs.Insert(ctx, amm.Pair, *updatedAMM)

	k.OnSwapEnd(ctx, *updatedAMM, quoteAssetAmt, baseAssetDelta, dir)

	return updatedAMM, baseAssetDelta, nil
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
func (k Keeper) SwapBaseAsset(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	dir v2types.Direction,
	baseAssetAmt sdk.Dec,
	quoteAssetLimit sdk.Dec,
) (updatedAMM *v2types.AMM, quoteAssetDelta sdk.Dec, err error) {
	if baseAssetAmt.IsZero() {
		return &amm, sdk.ZeroDec(), nil
	}

	if _, err = k.OracleKeeper.GetExchangeRate(ctx, amm.Pair); err != nil {
		return nil, sdk.Dec{}, v2types.ErrNoValidPrice.Wrapf("%s", amm.Pair)
	}

	updatedAMM, quoteAssetDelta, err = amm.SwapBaseAsset(baseAssetAmt, dir)
	if err != nil {
		return nil, sdk.Dec{}, err
	}

	if err := checkUserLimits(quoteAssetLimit, quoteAssetDelta, dir); err != nil {
		return nil, sdk.Dec{}, err
	}

	k.AMMs.Insert(ctx, amm.Pair, *updatedAMM)

	k.OnSwapEnd(ctx, *updatedAMM, quoteAssetDelta, baseAssetAmt, dir)

	return updatedAMM, quoteAssetDelta, err
}

// OnSwapEnd recalculates perp metrics for a particular pair.
func (k Keeper) OnSwapEnd(
	ctx sdk.Context,
	amm v2types.AMM,
	quoteAssetAmt sdk.Dec,
	baseAssetAmt sdk.Dec,
	dir v2types.Direction,
) {
	// Update Metrics
	metrics := k.Metrics.GetOr(ctx, amm.Pair, v2types.Metrics{
		Pair:        amm.Pair,
		NetSize:     sdk.ZeroDec(),
		VolumeQuote: sdk.ZeroDec(),
		VolumeBase:  sdk.ZeroDec(),
	})
	if dir == v2types.Direction_LONG {
		metrics.NetSize = metrics.NetSize.Add(baseAssetAmt)
	} else if dir == v2types.Direction_SHORT {
		metrics.NetSize = metrics.NetSize.Sub(baseAssetAmt)
	}
	metrics.VolumeBase = metrics.VolumeBase.Add(baseAssetAmt.Abs())
	metrics.VolumeQuote = metrics.VolumeQuote.Add(quoteAssetAmt.Abs())

	k.Metrics.Insert(ctx, amm.Pair, metrics)

	// -------------------- Emit events
	_ = ctx.EventManager().EmitTypedEvent(&v2types.MarkPriceChangedEvent{
		Pair:           amm.Pair,
		Price:          amm.MarkPrice(),
		BlockTimestamp: ctx.BlockTime(),
	})

	// TODO(k-yang): fix swap event values
	// _ = ctx.EventManager().EmitTypedEvent(&v2types.SwapEvent{
	// 	Pair:        amm.Pair,
	// 	QuoteAmount: quoteAssetAmt,
	// 	BaseAmount:  baseAssetAmt,
	// })
}
