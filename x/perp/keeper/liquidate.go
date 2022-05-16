package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vtypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type LiquidationOutput struct {
	FeeToPerpEcosystemFund sdk.Dec
	BadDebt                sdk.Dec
	FeeToLiquidator        sdk.Dec
	PositionResp           *types.PositionResp
}

func (l *LiquidationOutput) String() string {
	return fmt.Sprintf(`
	liquidationOutput {
		FeeToPerpEcosystemFund: %v,
		BadDebt: %v,
		FeeToLiquidator: %v,
		PositionResp: %v,
	}
	`,
		l.FeeToPerpEcosystemFund,
		l.BadDebt,
		l.FeeToLiquidator,
		l.PositionResp,
	)
}

func (l *LiquidationOutput) Validate() error {
	for _, field := range []sdk.Dec{
		l.FeeToPerpEcosystemFund, l.BadDebt, l.FeeToLiquidator} {
		if field.IsNil() {
			return fmt.Errorf(
				`invalid liquidationOutput: %v,
				must not have nil fields`, l.String())
		}
	}
	return nil
}

/* Liquidate allows to liquidate the trader position if the margin is below the
required margin maintenance ratio.
*/
func (k Keeper) Liquidate(
	ctx sdk.Context, pair common.TokenPair, trader sdk.AccAddress, liquidator sdk.AccAddress,
) error {
	// ------------- Liquidation Message Setup -------------

	// TODO validate liquidator (msg.Sender)
	// TODO validate trader (msg.PositionOwner)
	// TODO validate pair (msg.Vpool)
	position, err := k.GetPosition(ctx, pair, trader.String())
	if err != nil {
		return err
	}

	marginRatio, err := k.GetMarginRatio(ctx, *position, types.MarginCalculationPriceOption_MAX_PNL)
	if err != nil {
		return err
	}

	if k.VpoolKeeper.IsOverSpreadLimit(ctx, pair) {
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
		return types.MarginHighEnough
	}

	marginRatioBasedOnSpot, err := k.GetMarginRatio(
		ctx, *position, types.MarginCalculationPriceOption_SPOT)
	if err != nil {
		return err
	}

	fmt.Println("marginRatioBasedOnSpot", marginRatioBasedOnSpot)

	var (
		liquidationOutput LiquidationOutput
	)

	if marginRatioBasedOnSpot.GTE(params.GetPartialLiquidationRatioAsDec()) {
		liquidationOutput, err = k.CreatePartialLiquidation(ctx, pair, trader, position)
		if err != nil {
			return err
		}
	} else {
		liquidationOutput, err = k.CreateLiquidation(ctx, pair, trader, position)
		if err != nil {
			return err
		}
	}

	// Transfer fee from (which one?) module to PerpEF
	if liquidationOutput.FeeToPerpEcosystemFund.TruncateInt().IsPositive() {
		coinToPerpEF := sdk.NewCoin(
			pair.GetQuoteTokenDenom(), liquidationOutput.FeeToPerpEcosystemFund.TruncateInt())
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.VaultModuleAccount,
			types.PerpEFModuleAccount,
			sdk.NewCoins(coinToPerpEF),
		)
		if err != nil {
			return err
		}
	}

	// Transfer fee from (which one?) module to liquidator
	coinToLiquidator := sdk.NewCoin(
		pair.GetQuoteTokenDenom(), liquidationOutput.FeeToLiquidator.TruncateInt())
	err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		types.PerpEFModuleAccount,
		liquidator,
		sdk.NewCoins(coinToLiquidator),
	)
	if err != nil {
		panic(err) // Money for us
	}

	events.EmitPositionLiquidate(
		/* ctx */ ctx,
		/* vpool */ pair.String(),
		/* owner */ trader,
		/* notional */ liquidationOutput.PositionResp.ExchangedQuoteAssetAmount,
		/* vsize */ liquidationOutput.PositionResp.ExchangedPositionSize,
		/* liquidator */ liquidator,
		/* liquidationFee */ liquidationOutput.FeeToLiquidator.TruncateInt(),
		/* badDebt */ liquidationOutput.BadDebt,
	)

	return nil
}

//CreateLiquidation create a liquidation of a position and compute the fee to ecosystem fund
func (k Keeper) CreateLiquidation(
	ctx sdk.Context, pair common.TokenPair, owner sdk.AccAddress, position *types.Position,
) (liquidationOutput LiquidationOutput, err error) {
	params := k.GetParams(ctx)

	liquidationOutput.PositionResp, err = k.closePositionEntirely(ctx, *position, sdk.ZeroDec())

	if err != nil {
		return LiquidationOutput{}, err
	}

	remainMargin := liquidationOutput.PositionResp.MarginToVault.Abs()

	feeToLiquidator := liquidationOutput.PositionResp.ExchangedQuoteAssetAmount.
		Mul(params.GetLiquidationFeeAsDec()).
		Quo(sdk.MustNewDecFromStr("2"))
	totalBadDebt := liquidationOutput.PositionResp.BadDebt

	// if the remainMargin is not enough for liquidationFee, count it as bad debt
	// else, then the rest will be transferred to ecosystemFund
	if feeToLiquidator.GT(remainMargin) {
		liquidationBadDebt := feeToLiquidator.Sub(remainMargin)
		totalBadDebt = totalBadDebt.Add(liquidationBadDebt)
	} else {
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	if remainMargin.GT(sdk.ZeroDec()) {
		liquidationOutput.FeeToPerpEcosystemFund = remainMargin
	} else {
		liquidationOutput.FeeToPerpEcosystemFund = sdk.ZeroDec()
	}

	liquidationOutput.BadDebt = totalBadDebt
	liquidationOutput.FeeToLiquidator = feeToLiquidator

	err = liquidationOutput.Validate()
	if err != nil {
		return LiquidationOutput{}, err
	}
	return liquidationOutput, err
}

//CreatePartialLiquidation create a partial liquidation of a position and compute the fee to ecosystem fund
func (k Keeper) CreatePartialLiquidation(
	ctx sdk.Context, pair common.TokenPair, trader sdk.AccAddress, position *types.Position,
) (liquidationOutput LiquidationOutput, err error) {
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

	// half of the liquidationFee goes to liquidator & another half goes to ecosystem fund
	liquidationPenalty := positionResp.ExchangedQuoteAssetAmount.Mul(params.GetLiquidationFeeAsDec())
	feeToLiquidator := liquidationPenalty.Quo(sdk.MustNewDecFromStr("2"))

	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationPenalty)
	k.SetPosition(ctx, pair, trader.String(), positionResp.Position)

	liquidationOutput.PositionResp = positionResp

	liquidationOutput.FeeToPerpEcosystemFund = liquidationPenalty.Sub(feeToLiquidator)
	liquidationOutput.FeeToLiquidator = feeToLiquidator

	return
}
