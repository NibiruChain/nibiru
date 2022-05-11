package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vtypes "github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*Liquidate allows to liquidate the trader position if the margin is below the required margin maintenance ratio.*/
func (k Keeper) Liquidate(ctx sdk.Context, pair common.TokenPair, trader string, sender sdk.AccAddress) error {
	var (
		feeToInsuranceFund sdk.Int
		liquidationPenalty sdk.Int
		isPartialClose     bool
	)

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

	marginRatioBasedOnSpot, err := k.GetMarginRatio(ctx, pair, trader, types.MarginCalculationPriceOption_SPOT)
	if err != nil {
		return err
	}

	// Liquidate position
	position, err := k.GetPosition(ctx, pair, trader)
	if err != nil {
		return err
	}

	if marginRatioBasedOnSpot.GTE(sdk.NewInt(params.PartialLiquidationRatio)) {
		feeToInsuranceFund, err = k.createPartialLiquidation(ctx, pair, trader, position)
		if err != nil {
			return err
		}
		isPartialClose = true
	} else {
		feeToInsuranceFund, liquidationPenalty, err = k.createLiquidation(ctx, pair, trader, position)
		if err != nil {
			return err
		}

		isPartialClose = false
	}

	if feeToInsuranceFund.GT(sdk.ZeroInt()) {
		k.BankKeeper.SendCoinsFromAccountToModule(ctx, trader, types.ModuleName, sdk.NewCoins(sdk.NewCoin(pair.GetQuoteTokenDenom(), feeToInsuranceFund)))
	}
	k.withdraw(ctx, pair.GetQuoteTokenDenom(), sender, feeToLiquidator)

	events.EmitPositionLiquidate(ctx, pair, trader, positionNotional, position.Size_, sender)

	return nil
}

//createLiquidation create a liquidation of a position and compute the fee to insurance fund
func (k Keeper) createLiquidation(ctx sdk.Context, pair common.TokenPair, trader string, position *types.Position) (
	feeToInsuranceFund sdk.Int, liquidationPenalty sdk.Int, err error) {
	params := k.GetParams(ctx)

	liquidationPenalty = position.Margin
	positionResp, err := k.closePosition(ctx, pair, trader, sdk.ZeroInt())

	if err != nil {
		return
	}

	remainMargin := positionResp.MarginToVault.Abs()

	feeToLiquidator := sdk.NewDecFromInt(positionResp.ExchangedQuoteAssetAmount).Mul(params.GetLiquidationFeeAsDec()).Quo(sdk.MustNewDecFromStr("2")).TruncateInt()
	totalBadDebt := positionResp.BadDebt

	// if the remainMargin is not enough for liquidationFee, count it as bad debt
	// else, then the rest will be transferred to insuranceFund
	if feeToLiquidator.GT(remainMargin) {
		liquidationBadDebt := feeToLiquidator.Sub(remainMargin)
		totalBadDebt = totalBadDebt.Add(liquidationBadDebt)
	} else {
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	// transfer the actual token between trader and vault
	if totalBadDebt.GT(sdk.ZeroInt()) {
		err = k.realizeBadDebt(ctx, pair.GetQuoteTokenDenom(), totalBadDebt)
		if err != nil {
			return
		}
	}
	if remainMargin.GT(sdk.ZeroInt()) {
		feeToInsuranceFund = remainMargin
	}

	return
}

//createPartialLiquidation create a partial liquidation of a position and compute the fee to insurance fund
func (k Keeper) createPartialLiquidation(ctx sdk.Context, pair common.TokenPair, trader string, position *types.Position) (feeToInsuranceFund sdk.Int, err error) {
	params := k.GetParams(ctx)
	var (
		dir  vtypes.Direction
		side types.Side
	)

	if position.Size_.GTE(sdk.ZeroInt()) {
		dir = vtypes.Direction_ADD_TO_POOL
		side = types.Side_SELL
	} else {
		dir = vtypes.Direction_REMOVE_FROM_POOL
		side = types.Side_BUY
	}

	partiallyLiquidatedPositionNotional, err := k.VpoolKeeper.GetOutputPrice(
		ctx,
		pair,
		dir,
		/*abs= */ sdk.NewDecFromInt(position.Size_).Mul(params.GetPartialLiquidationRatioAsDec()).Abs(),
	)
	if err != nil {
		return feeToInsuranceFund, err
	}

	positionResp, err := k.openReversePosition(
		/*ctx= */ ctx,
		/*pair= */ pair,
		/*side= */ side,
		/*trader= */ trader,
		/*quoteAssetAmount= */ partiallyLiquidatedPositionNotional.TruncateInt(),
		/*leverage= */ sdk.OneInt(),
		/*baseAssetAmountLimit= */ sdk.ZeroInt(),
		/*canOverFluctuationLimit= */ true,
	)
	if err != nil {
		return feeToInsuranceFund, err
	}

	// half of the liquidationFee goes to liquidator & another half goes to insurance fund
	liquidationPenalty := sdk.NewDecFromInt(positionResp.ExchangedQuoteAssetAmount).Mul(params.GetLiquidationFeeAsDec()).TruncateInt()
	feeToLiquidator := sdk.NewDecFromInt(liquidationPenalty).Quo(sdk.MustNewDecFromStr("2")).TruncateInt()

	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationPenalty)
	k.SetPosition(ctx, pair, trader, positionResp.Position)

	feeToInsuranceFund = liquidationPenalty.Sub(feeToLiquidator)
	return
}
