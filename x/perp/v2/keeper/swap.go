package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// checkUserLimits checks if the limit is violated by the amount.
// returns error if it does
func checkUserLimits(limit, amount sdk.Dec, dir types.Direction) error {
	if limit.IsZero() {
		return nil
	}

	if dir == types.Direction_LONG && amount.LT(limit) {
		return types.ErrAssetFailsUserLimit.Wrapf(
			"amount (%s) is less than selected limit (%s)",
			amount.String(),
			limit.String(),
		)
	}

	if dir == types.Direction_SHORT && amount.GT(limit) {
		return types.ErrAssetFailsUserLimit.Wrapf(
			"amount (%s) is greater than selected limit (%s)",
			amount.String(),
			limit.String(),
		)
	}

	return nil
}

// SwapQuoteAsset trades quoteAssets in exchange for baseAssets.
// Updates the AMM reserves and persists it to state.
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
//   - baseAssetDelta: the amount of base assets swapped, unsigned
//   - err: error if any
//
// NOTE: the baseAssetDelta is always positive
func (k Keeper) SwapQuoteAsset(
	ctx sdk.Context,
	market types.Market,
	amm types.AMM,
	dir types.Direction,
	quoteAssetAmt sdk.Dec, // unsigned
	baseAssetLimit sdk.Dec, // unsigned
) (updatedAMM *types.AMM, baseAssetDelta sdk.Dec, err error) {
	baseAssetDelta, err = amm.SwapQuoteAsset(quoteAssetAmt, dir)
	if err != nil {
		return nil, sdk.Dec{}, err
	}

	if err := checkUserLimits(baseAssetLimit, baseAssetDelta, dir); err != nil {
		return nil, sdk.Dec{}, err
	}

	k.SaveAMM(ctx, amm)

	return &amm, baseAssetDelta, nil
}

// SwapBaseAsset trades baseAssets in exchange for quoteAssets.
// Updates the AMM reserves and persists it to state.
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
	amm types.AMM,
	dir types.Direction,
	baseAssetAmt sdk.Dec,
	quoteAssetLimit sdk.Dec,
) (updatedAMM *types.AMM, quoteAssetDelta sdk.Dec, err error) {
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

	k.SaveAMM(ctx, amm)

	return &amm, quoteAssetDelta, err
}
