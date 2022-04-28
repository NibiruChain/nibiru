package stablecoin

import (
	"github.com/NibiruChain/nibiru/x/stablecoin/keeper"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker updates the current pricefeed
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	if !k.GetParams(ctx).IsCollateralRatioValid {
		// Try to re-start the collateral ratio updates
		err := k.EvaluateCollRatio(ctx)

		params := k.GetParams(ctx)
		if err != nil {
			k.SetParams(ctx, types.NewParams(
				params.GetCollRatioAsDec(),
				params.GetFeeRatioAsDec(),
				params.GetEfFeeRatioAsDec(),
				params.GetBonusRateRecollAsDec(),
				params.DistrEpochIdentifier,
				params.GetAdjustmentStepAsDec(),
				params.GetPriceLowerBoundAsDec(),
				params.GetPriceUpperBoundAsDec(),
				/*isCollateralRatioValid*/ false,
			))
			return
		}

		k.SetParams(ctx, types.NewParams(
			params.GetCollRatioAsDec(),
			params.GetFeeRatioAsDec(),
			params.GetEfFeeRatioAsDec(),
			params.GetBonusRateRecollAsDec(),
			params.DistrEpochIdentifier,
			params.GetAdjustmentStepAsDec(),
			params.GetPriceLowerBoundAsDec(),
			params.GetPriceUpperBoundAsDec(),
			/*isCollateralRatioValid*/ true,
		))
	}
}
