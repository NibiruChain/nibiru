package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

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

// SwapQuoteAsset trades quoteAssets in exchange for baseAssets.
// The base asset is a crypto asset like BTC.
// The quote asset is a stablecoin like NUSD.
//
// args:
//   - ctx: cosmos-sdk context
//   - market: a market like BTC:NUSD
//   - amm: the reserves of the AMM
//   - dir: the direction the user takes
//   - quoteAssetAmt: the amount of quote assets to swap, must be positive
//   - baseAssetLimit: the limit of base assets to swap
//
// returns:
//   - updatedAMM: the updated amm
//   - baseAssetDelta: the amount of base assets swapped
//   - err: error if any
//
// NOTE: the baseAssetDelta is always positive
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

	baseAssetDelta, err = amm.SwapQuoteAsset(quoteAssetAmt, dir)
	if err != nil {
		return nil, sdk.Dec{}, err
	}

	if err := checkUserLimits(baseAssetLimit, baseAssetDelta, dir); err != nil {
		return nil, sdk.Dec{}, err
	}

	k.AMMs.Insert(ctx, amm.Pair, amm)

	return &amm, baseAssetDelta, nil
}

// SwapBaseAsset trades baseAssets in exchange for quoteAssets.
// The base asset is a crypto asset like BTC.
// The quote asset is a stablecoin like NUSD.
//
// args:
//   - ctx: cosmos-sdk context
//   - market: a market like BTC:NUSD
//   - amm: the reserves of the AMM
//   - dir: the direction the user takes
//   - baseAssetAmt: the amount of base assets to swap, must be positive
//   - quoteAssetLimit: the limit of quote assets to swap
//
// returns:
//   - updatedAMM: the updated amm
//   - quoteAssetDelta: the amount of quote assets swapped
//   - err: error if any
//
// NOTE: the quoteAssetDelta is always positive
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

	quoteAssetDelta, err = amm.SwapBaseAsset(baseAssetAmt, dir)
	if err != nil {
		return nil, sdk.Dec{}, err
	}

	if err := checkUserLimits(quoteAssetLimit, quoteAssetDelta, dir); err != nil {
		return nil, sdk.Dec{}, err
	}

	k.AMMs.Insert(ctx, amm.Pair, amm)

	return &amm, quoteAssetDelta, err
}
