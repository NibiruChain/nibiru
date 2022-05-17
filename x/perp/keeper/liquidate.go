package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
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

// CreateLiquidation create a liquidation of a position and compute the fee to ecosystem fund
func (k Keeper) CreateLiquidation(
	ctx sdk.Context, pair common.TokenPair, owner sdk.AccAddress, position *types.Position,
) (LiquidationOutput, error) {
	params := k.GetParams(ctx)

	positionResp, err := k.closePositionEntirely(ctx, *position, sdk.ZeroDec())
	if err != nil {
		return LiquidationOutput{}, err
	}

	remainMargin := positionResp.MarginToVault.Abs()

	feeToLiquidator := positionResp.ExchangedQuoteAssetAmount.
		Mul(params.GetLiquidationFeeAsDec()).
		Quo(sdk.MustNewDecFromStr("2"))
	totalBadDebt := positionResp.BadDebt

	if feeToLiquidator.GT(remainMargin) {
		// if the remainMargin is not enough for liquidationFee, count it as bad debt
		liquidationBadDebt := feeToLiquidator.Sub(remainMargin)
		totalBadDebt = totalBadDebt.Add(liquidationBadDebt)
	} else {
		// Otherwise, the remaining margin rest will be transferred to ecosystemFund
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	var feeToPerpEcosystemFund sdk.Dec
	if remainMargin.GT(sdk.ZeroDec()) {
		feeToPerpEcosystemFund = remainMargin
	} else {
		feeToPerpEcosystemFund = sdk.ZeroDec()
	}

	output := LiquidationOutput{
		FeeToPerpEcosystemFund: feeToPerpEcosystemFund,
		BadDebt:                totalBadDebt,
		FeeToLiquidator:        feeToLiquidator,
		PositionResp:           positionResp,
	}

	err = output.Validate()
	if err != nil {
		return LiquidationOutput{}, err
	}
	return output, err
}
