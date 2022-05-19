package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vtypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type LiquidateResp struct {
	BadDebt                sdk.Dec
	FeeToLiquidator        sdk.Dec
	FeeToPerpEcosystemFund sdk.Dec
	Liquidator             sdk.AccAddress
	PositionResp           *types.PositionResp
}

func (l *LiquidateResp) String() string {
	return fmt.Sprintf(`
	LiquidateResp {
		BadDebt: %v,
		FeeToLiquidator: %v,
		FeeToPerpEcosystemFund: %v,
		PositionResp: %v,
		Liquidator: %v,
	}
	`,
		l.BadDebt.String(),
		l.FeeToLiquidator.String(),
		l.FeeToPerpEcosystemFund.String(),
		l.PositionResp.String(),
		l.Liquidator.String(),
	)
}

func (l *LiquidateResp) Validate() error {
	for _, field := range []sdk.Dec{
		l.BadDebt, l.FeeToLiquidator, l.FeeToPerpEcosystemFund} {
		if field.IsNil() {
			return fmt.Errorf(
				`invalid liquidationOutput: %v,
				must not have nil fields`, l.String())
		}
	}
	return nil
}

// Liquidate liquidates check the margin and either liquidate or partially liquidate a position
// TODO: Change inputs of the function to take a MsgLiquidate input and return a MsgLiquidateOutput
func (k Keeper) Liquidate(ctx sdk.Context, liquidator sdk.AccAddress, position *types.Position) (err error) {
	marginRatio, err := k.GetMarginRatio(ctx, *position, types.MarginCalculationPriceOption_MAX_PNL)
	if err != nil {
		return err
	}

	if k.VpoolKeeper.IsOverSpreadLimit(ctx, common.TokenPair(position.GetPair())) {
		marginRatioBasedOnOracle, err := k.GetMarginRatio(
			ctx, *position, types.MarginCalculationPriceOption_INDEX)
		if err != nil {
			return err
		}

		marginRatio = sdk.MaxDec(marginRatio, marginRatioBasedOnOracle)
	}

	params := k.GetParams(ctx)
	err = requireMoreMarginRatio(marginRatio, params.MaintenanceMarginRatio, false)

	if err != nil {
		return types.ErrMarginHighEnough
	}

	marginRatioBasedOnSpot, err := k.GetMarginRatio(
		ctx, *position, types.MarginCalculationPriceOption_SPOT)
	if err != nil {
		return err
	}

	if marginRatioBasedOnSpot.GTE(params.GetPartialLiquidationRatioAsDec()) {
		err = k.ExecutePartialLiquidation(ctx, liquidator, position)
	} else {
		err = k.ExecuteFullLiquidation(ctx, liquidator, position)
	}
	return
}

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

	err = k.distributeLiquidateRewards(ctx, LiquidateResp{
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

// ExecutePartialLiquidation partially liquidates a position
func (k Keeper) ExecutePartialLiquidation(ctx sdk.Context, liquidator sdk.AccAddress, position *types.Position) (err error) {
	params := k.GetParams(ctx)

	var dir vtypes.Direction

	if position.Size_.GTE(sdk.ZeroDec()) {
		dir = vtypes.Direction_ADD_TO_POOL
	} else {
		dir = vtypes.Direction_REMOVE_FROM_POOL
	}

	partiallyLiquidatedPositionNotional, err := k.VpoolKeeper.GetBaseAssetPrice(
		ctx,
		common.TokenPair(position.Pair),
		dir,
		/* abs= */ position.Size_.Mul(params.GetPartialLiquidationRatioAsDec().Abs()),
	)
	if err != nil {
		return
	}

	positionResp, err := k.openReversePosition(
		/* ctx */ ctx,
		/* currentPosition */ *position,
		/* quoteAssetAmount */ partiallyLiquidatedPositionNotional,
		/* leverage */ sdk.OneDec(),
		/* baseAssetAmountLimit */ sdk.ZeroDec(),
		/* canOverFluctuationLimit */ true,
	)
	if err != nil {
		return
	}

	// half of the liquidationFee goes to liquidator & another half goes to ecosystem fund
	liquidationPenalty := positionResp.ExchangedQuoteAssetAmount.Mul(params.GetLiquidationFeeAsDec())
	feeToLiquidator := liquidationPenalty.Quo(sdk.MustNewDecFromStr("2"))

	// Remove the liquidation penalty from the margin of the position
	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationPenalty)
	k.SetPosition(ctx, common.TokenPair(position.Pair), position.Address, positionResp.Position)

	err = k.distributeLiquidateRewards(ctx, LiquidateResp{
		BadDebt:                sdk.ZeroDec(),
		FeeToLiquidator:        feeToLiquidator,
		FeeToPerpEcosystemFund: liquidationPenalty.Sub(feeToLiquidator),
		Liquidator:             liquidator,
		PositionResp:           positionResp,
	})

	return
}

func (k Keeper) distributeLiquidateRewards(
	ctx sdk.Context, liquidateResp LiquidateResp) (err error) {
	// --------------------------------------------------------------
	//  Preliminary validations
	// --------------------------------------------------------------

	// validate response
	err = liquidateResp.Validate()
	if err != nil {
		return err
	}

	// validate liquidator
	liquidator, err := sdk.AccAddressFromBech32(liquidateResp.Liquidator.String())
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
			/* to */ liquidator,
			sdk.NewCoins(coinToLiquidator),
		)
		if err != nil {
			return err
		}
		events.EmitTransfer(ctx,
			/* coin */ coinToLiquidator,
			/* from */ perpEFAddr.String(),
			/* to */ liquidator.String(),
		)
	}

	return nil
}
