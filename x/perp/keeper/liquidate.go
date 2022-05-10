package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*Liquidate allows to liquidate the trader position if the margin is below the required margin maintenance ratio.*/
func (k Keeper) Liquidate(ctx sdk.Context, pair common.TokenPair, trader string) (sdk.Int, error) {
	marginRatio, err := k.GetMarginRatio(ctx, pair, trader, types.MarginCalculationPriceOption_MAX_PNL)
	if err != nil {
		return sdk.NewInt(0), err
	}

	if pair.IsOverSpreadLimit(ctx) {
		marginRatioBasedOnOracle, err := k.GetMarginRatio(ctx, pair, trader, types.MarginCalculationPriceOption_INDEX)
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

	marginRatioBasedOnSpot, err := k.GetMarginRatio(ctx, pair, trader, types.MarginCalculationPriceOption_SPOT)

	if marginRatioBasedOnSpot.GTE(sdk.NewInt(params.LiquidationFee)) {

	}
}
