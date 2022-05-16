package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vtypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

/*
WIP, missing items:

Dev:
	- [] realizeBadDebt

Tests:
	- [] CreateLiquidation and createPartialLiquidation
	- [] Liquidation
	- [] IsOverSpreadLimit
*/

type LiquidationOutput struct {
	FeeToInsuranceFund sdk.Dec
	BadDebt            sdk.Dec
	FeeToLiquidator    sdk.Dec
	PositionResp       *types.PositionResp
}

/*Liquidate allows to liquidate the trader position if the margin is below the required margin maintenance ratio.*/
func (k Keeper) Liquidate(ctx sdk.Context, pair common.TokenPair, trader sdk.AccAddress, liquidator sdk.AccAddress) error {
	// Liquidate position
	owner := trader.String()
	position, err := k.GetPosition(ctx, pair, trader.String())
	if err != nil {
		panic(err)
	}

	marginRatio, err := k.GetMarginRatio(ctx, *position, types.MarginCalculationPriceOption_MAX_PNL)
	if err != nil {
		return err
	}

	if k.VpoolKeeper.IsOverSpreadLimit(ctx, pair) {
		marginRatioBasedOnOracle, err := k.GetMarginRatio(ctx, *position, types.MarginCalculationPriceOption_INDEX)
		if err != nil {
			return err
		}

		marginRatio = sdk.MaxDec(marginRatio, marginRatioBasedOnOracle)
	}

	params := k.GetParams(ctx)
	err = requireMoreMarginRatio(marginRatio, params.MaintenanceMarginRatio, false)
	if err != nil {
		return types.MarginHighEnough
	}

	marginRatioBasedOnSpot, err := k.GetMarginRatio(ctx, *position, types.MarginCalculationPriceOption_SPOT)
	if err != nil {
		return err
	}

	fmt.Println("marginRatioBasedOnSpot", marginRatioBasedOnSpot)

	var (
		liquidationOuptut LiquidationOutput
	)

	if marginRatioBasedOnSpot.GTE(params.GetPartialLiquidationRatioAsDec()) {
		liquidationOuptut, err = k.createPartialLiquidation(ctx, pair, trader, position)
		if err != nil {
			panic(err)
		}
	} else {
		liquidationOuptut, err = k.CreateLiquidation(ctx, pair, trader, position)
		if err != nil {
			panic(err)
		}
	}

	err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx,
		trader,
		common.TreasuryPoolModuleAccount,
		sdk.NewCoins(sdk.NewCoin(pair.GetQuoteTokenDenom(), liquidationOuptut.FeeToInsuranceFund.TruncateInt())),
	)
	if err != nil {
		panic(err)
	}
	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		common.TreasuryPoolModuleAccount,
		liquidator,
		sdk.NewCoins(sdk.NewCoin(pair.GetQuoteTokenDenom(), liquidationOuptut.FeeToLiquidator.TruncateInt())),
	)
	if err != nil {
		panic(err)
	}

	events.EmitPositionLiquidate(
		/* ctx */ ctx,
		/* vpool */ pair.String(),
		/* owner */ owner,
		/* notional */ liquidationOuptut.PositionResp.ExchangedQuoteAssetAmount,
		/* vsize */ liquidationOuptut.PositionResp.ExchangedPositionSize,
		/* liquidator */ liquidator,
		/* liquidationFee */ liquidationOuptut.FeeToLiquidator.TruncateInt(),
		/* badDebt */ liquidationOuptut.BadDebt,
	)

	return nil
}

//CreateLiquidation create a liquidation of a position and compute the fee to insurance fund
func (k Keeper) CreateLiquidation(ctx sdk.Context, pair common.TokenPair, owner sdk.AccAddress, position *types.Position) (
	liquidationOutput LiquidationOutput, err error) {
	params := k.GetParams(ctx)

	liquidationOutput.PositionResp, err = k.closePositionEntirely(ctx, *position, sdk.ZeroDec())

	if err != nil {
		return
	}

	remainMargin := liquidationOutput.PositionResp.MarginToVault.Abs()

	feeToLiquidator := liquidationOutput.PositionResp.ExchangedQuoteAssetAmount.Mul(params.GetLiquidationFeeAsDec()).Quo(sdk.MustNewDecFromStr("2"))
	totalBadDebt := liquidationOutput.PositionResp.BadDebt

	// if the remainMargin is not enough for liquidationFee, count it as bad debt
	// else, then the rest will be transferred to insuranceFund
	if feeToLiquidator.GT(remainMargin) {
		liquidationBadDebt := feeToLiquidator.Sub(remainMargin)
		totalBadDebt = totalBadDebt.Add(liquidationBadDebt)
		fmt.Println("liquidationBadDebt", liquidationBadDebt)
	} else {
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	if remainMargin.GT(sdk.ZeroDec()) {
		liquidationOutput.FeeToInsuranceFund = remainMargin
	}

	liquidationOutput.BadDebt = totalBadDebt
	liquidationOutput.FeeToLiquidator = feeToLiquidator

	return
}

//createPartialLiquidation create a partial liquidation of a position and compute the fee to insurance fund
func (k Keeper) createPartialLiquidation(ctx sdk.Context, pair common.TokenPair, trader sdk.AccAddress, position *types.Position) (liquidationOutput LiquidationOutput, err error) {
	params := k.GetParams(ctx)
	var (
		dir vtypes.Direction
	)

	if position.Size_.GTE(sdk.ZeroDec()) {
		dir = vtypes.Direction_ADD_TO_POOL
	} else {
		dir = vtypes.Direction_REMOVE_FROM_POOL
	}

	partiallyLiquidatedPositionNotional, err := k.VpoolKeeper.GetBaseAssetPrice(
		ctx,
		pair,
		dir,
		/*abs= */ position.Size_.Mul(params.GetPartialLiquidationRatioAsDec()).Abs(),
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

	// half of the liquidationFee goes to liquidator & another half goes to insurance fund
	liquidationPenalty := positionResp.ExchangedQuoteAssetAmount.Mul(params.GetLiquidationFeeAsDec())
	feeToLiquidator := liquidationPenalty.Quo(sdk.MustNewDecFromStr("2"))

	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationPenalty)
	k.SetPosition(ctx, pair, trader.String(), positionResp.Position)

	liquidationOutput.FeeToInsuranceFund = liquidationPenalty.Sub(feeToLiquidator)
	liquidationOutput.PositionResp = positionResp
	liquidationOutput.FeeToLiquidator = feeToLiquidator

	return
}
