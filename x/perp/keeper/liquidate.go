package keeper

import (
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) Liquidate(ctx sdk.Context, amm types.IVirtualPool, trader string) (sdk.Int, error) {
	marginRatio, err := k.GetMarginRatio(ctx, amm, trader, types.MarginCalculationPriceOption_SPOT)
	if err != nil {
		return sdk.NewInt(0), err
	}

	if amm.IsOverSpreadLimit(ctx) {
		marginRatioBasedOnOracle, err := k.GetMarginRatio(ctx, amm, trader, types.MarginCalculationPriceOption_INDEX)
		if err != nil {
			return sdk.NewInt(0), err
		}

		marginRatio = sdk.MaxInt(marginRatio, marginRatioBasedOnOracle)
	}

	params := k.GetParams(ctx)
	if marginRatio.GTE(params.MaintenanceMarginRatio) {
		return sdk.NewInt(0), types.MarginHighEnough
	}

	// Liquidate position

	marginRatioBasedOnSpot, err := k.GetMarginRatio(ctx, amm, trader, types.MarginCalculationPriceOption_SPOT)

	if marginRatioBasedOnSpot.GTE(params.LiquidationFee)
}
