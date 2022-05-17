package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

type LiquidateResp struct {
	BadDebt                sdk.Int
	FeeToLiquidator        sdk.Int
	FeeToPerpEcosystemFund sdk.Int
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
		l.PositionResp,
		l.Liquidator.String(),
	)
}

func (l *LiquidateResp) Validate() error {
	for _, field := range []sdk.Int{
		l.BadDebt, l.FeeToLiquidator, l.FeeToPerpEcosystemFund} {
		if field.IsNil() {
			return fmt.Errorf(
				`invalid liquidationOutput: %v,
				must not have nil fields`, l.String())
		}
	}
	return nil
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

	// NOTE  following prints should be removed
	fmt.Println(positionResp.String())
	fmt.Println("exchanged pos size:", positionResp.ExchangedPositionSize.String())
	fmt.Println("exchange quote amt:", positionResp.ExchangedQuoteAssetAmount.String())
	fmt.Println("margin to vault:", positionResp.MarginToVault.String())

	feeToLiquidator := params.GetLiquidationFeeAsDec().
		MulInt(positionResp.ExchangedQuoteAssetAmount).
		QuoInt64(2).TruncateInt()
	totalBadDebt := positionResp.BadDebt

	if feeToLiquidator.GT(remainMargin) {
		// if the remainMargin is not enough for liquidationFee, count it as bad debt
		liquidationBadDebt := feeToLiquidator.Sub(remainMargin)
		totalBadDebt = totalBadDebt.Add(liquidationBadDebt)
	} else {
		// Otherwise, the remaining margin rest will be transferred to ecosystemFund
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	feeToPerpEcosystemFund := sdk.ZeroInt()
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
	feeToPerpEF := liquidateResp.FeeToPerpEcosystemFund
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
	feeToLiquidator := liquidateResp.FeeToLiquidator
	if feeToLiquidator.IsPositive() {
		coinToLiquidator := sdk.NewCoin(
			pair.GetQuoteTokenDenom(), liquidateResp.FeeToLiquidator)
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
