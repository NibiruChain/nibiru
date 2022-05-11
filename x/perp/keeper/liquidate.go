package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vtypes "github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*Liquidate allows to liquidate the trader position if the margin is below the required margin maintenance ratio.*/
func (k Keeper) Liquidate(ctx sdk.Context, pair common.TokenPair, trader string) error {
	marginRatio, err := k.GetMarginRatio(ctx, pair, trader, types.MarginCalculationPriceOption_MAX_PNL)
	if err != nil {
		return err
	}

	if k.VpoolKeeper.IsOverSpreadLimit(ctx, pair) {
		marginRatioBasedOnOracle, err := k.GetMarginRatio(ctx, pair, trader, types.MarginCalculationPriceOption_INDEX)
		if err != nil {
			return err
		}

		marginRatio = sdk.MaxInt(marginRatio, marginRatioBasedOnOracle)
	}

	params := k.GetParams(ctx)

	err = requireMoreMarginRatio(marginRatio, params.MaintenanceMarginRatio, false)
	if err != nil {
		return types.MarginHighEnough
	}

	// Liquidate position
	var position *types.Position
	position, err = k.GetPosition(ctx, pair, trader)
	if err != nil {
		return err
	}

	/*
		liquidationPenalty = getPosition(_amm, _trader).margin;
		positionResp = internalClosePosition(_amm, _trader, Decimal.zero());
		Decimal.decimal memory remainMargin = positionResp.marginToVault.abs();
		feeToLiquidator = positionResp.exchangedQuoteAssetAmount.mulD(liquidationFeeRatio).divScalar(2);
	*/

	liquidationPenalty := position.Margin
	positionResp, resp := k.closePosition(ctx, pair, trader, sdk.ZeroInt())
	remainMargin := positionResp.MarginToVault.Abs()
	feeToLiquidator := positionResp.ExchangedQuoteAssetAmount.Mul(params.GetLiquidationFeeAsDec().RoundInt()).Quo(sdk.NewInt(2))

	marginRatioBasedOnSpot, err := k.GetMarginRatio(ctx, pair, trader, types.MarginCalculationPriceOption_SPOT)
	if err != nil {
		return err
	}

	if marginRatioBasedOnSpot.GTE(sdk.NewInt(params.LiquidationFee)) {
		var dir vtypes.Direction

		if position.Size_.GTE(sdk.ZeroInt()) {
			dir = vtypes.Direction_ADD_TO_POOL
		} else {
			dir = vtypes.Direction_REMOVE_FROM_POOL
		}

		partiallyLiquidatedPositionNotional, err := k.VpoolKeeper.GetOutputPrice(
			ctx, pair, dir, position.Size_.Mul(params.PartialLiquidationRatio).Abs(),
		)

		if err != nil {
			return err
		}
	}
}
