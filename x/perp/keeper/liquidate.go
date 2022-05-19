package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// ExecuteFullLiquidation fully liquidates a position.
func (k Keeper) ExecuteFullLiquidation(
	ctx sdk.Context, liquidator sdk.AccAddress, position *types.Position,
) (err error) {
	params := k.GetParams(ctx)

	positionResp, err := k.closePositionEntirely(
		ctx,
		/* currentPosition */ *position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec())
	if err != nil {
		return err
	}

	remainMargin := positionResp.MarginToVault.Abs()

	feeToLiquidator := params.GetLiquidationFeeAsDec().
		Mul(positionResp.ExchangedQuoteAssetAmount).
		QuoInt64(2)
	totalBadDebt := positionResp.BadDebt

	if feeToLiquidator.GT(remainMargin) {
		// if the remainMargin is not enough for liquidationFee, count it as bad debt
		liquidationBadDebt := feeToLiquidator.Sub(remainMargin)
		totalBadDebt = totalBadDebt.Add(liquidationBadDebt)
	} else {
		// Otherwise, the remaining margin rest will be transferred to ecosystemFund
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	feeToPerpEcosystemFund := sdk.ZeroDec()
	if remainMargin.IsPositive() {
		feeToPerpEcosystemFund = remainMargin
	}

	err = k.distributeLiquidateRewards(ctx, types.LiquidateResp{
		BadDebt:                totalBadDebt,
		FeeToLiquidator:        feeToLiquidator,
		FeeToPerpEcosystemFund: feeToPerpEcosystemFund,
		Liquidator:             liquidator,
		PositionResp:           positionResp,
	})
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) distributeLiquidateRewards(
	ctx sdk.Context, liquidateResp types.LiquidateResp) (err error) {
	// --------------------------------------------------------------
	//  Preliminary validations
	// --------------------------------------------------------------

	// validate response
	err = liquidateResp.Validate()
	if err != nil {
		return err
	}

	// validate pair
	pair, err := common.NewTokenPairFromStr(liquidateResp.PositionResp.Position.Pair)
	if err != nil {
		return err
	}
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return err
	}

	// --------------------------------------------------------------
	// Distribution of rewards
	// --------------------------------------------------------------

	vaultAddr := k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount)
	perpEFAddr := k.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount)

	// Transfer fee from vault to PerpEF
	feeToPerpEF := liquidateResp.FeeToPerpEcosystemFund.RoundInt()
	if feeToPerpEF.IsPositive() {
		coinToPerpEF := sdk.NewCoin(
			pair.GetQuoteTokenDenom(), feeToPerpEF)
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			/* from */ types.VaultModuleAccount,
			/* to */ types.PerpEFModuleAccount,
			sdk.NewCoins(coinToPerpEF),
		)
		if err != nil {
			return err
		}
		events.EmitTransfer(ctx,
			/* coin */ coinToPerpEF,
			/* from */ vaultAddr.String(),
			/* to */ perpEFAddr.String(),
		)
	}

	// Transfer fee from PerpEF to liquidator
	feeToLiquidator := liquidateResp.FeeToLiquidator.RoundInt()
	if feeToLiquidator.IsPositive() {
		coinToLiquidator := sdk.NewCoin(
			pair.GetQuoteTokenDenom(), feeToLiquidator)
		err = k.BankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			/* from */ types.PerpEFModuleAccount,
			/* to */ liquidateResp.Liquidator,
			sdk.NewCoins(coinToLiquidator),
		)
		if err != nil {
			return err
		}
		events.EmitTransfer(ctx,
			/* coin */ coinToLiquidator,
			/* from */ perpEFAddr.String(),
			/* to */ liquidateResp.Liquidator.String(),
		)
	}

	return nil
}
