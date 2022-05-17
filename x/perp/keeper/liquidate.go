package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
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

/* CreatePartialLiquidation returns the 'LiquidationOutput' of a partial liquidation.

Args:
- ctx (sdk.Context): Carries information about the current state of the application.
- pair (common.TokenPair): identifier for the virtual pool
- trader (sdk.AccAddress): address of the owner of the position
- position: the position that is will be partially liquidated

Returns:
- (*LiquidationOutput): fees, bad debt, and position response for the partial liquidation
- (error): An error if one is raised.
*/
func (k Keeper) CreatePartialLiquidation(
	ctx sdk.Context,
	pair common.TokenPair,
	trader sdk.AccAddress,
	position *types.Position,
) (*LiquidationOutput, error) {

	// Get position direction: long or short
	var (
		dir vpooltypes.Direction
	)
	if position.Size_.GTE(sdk.ZeroDec()) {
		dir = vpooltypes.Direction_ADD_TO_POOL
	} else {
		dir = vpooltypes.Direction_REMOVE_FROM_POOL
	}

	// Compute the notional of the portion of position that's being liquidated
	params := k.GetParams(ctx)
	partiallyLiquidatedPositionNotional, err := k.VpoolKeeper.GetBaseAssetPrice(
		ctx, pair, dir,
		/* baseAssetAmount */ position.Size_.Mul(params.GetPartialLiquidationRatioAsDec()).Abs(),
	)
	if err != nil {
		return nil, err
	}

	// Partially close (i.e. open reverse) the position
	positionResp, err := k.openReversePosition(
		/* ctx */ ctx,
		/* currentPosition */ *position,
		/* quoteAssetAmount */ partiallyLiquidatedPositionNotional,
		/* leverage */ sdk.OneDec(),
		/* baseAssetAmountLimit */ sdk.ZeroDec(),
		/* canOverFluctuationLimit */ true,
	)
	if err != nil {
		return nil, err
	}

	// Compute the liquidation penality, of which half goes to the liquidator
	// and half goes to the ecosystem fund.
	liquidationPenalty := positionResp.ExchangedQuoteAssetAmount.Mul(params.GetLiquidationFeeAsDec())
	feeToLiquidator := liquidationPenalty.Quo(sdk.MustNewDecFromStr("2"))
	feeToPerpEF := liquidationPenalty.Sub(feeToLiquidator)

	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationPenalty)
	k.SetPosition(ctx, pair, trader.String(), positionResp.Position)

	return &LiquidationOutput{
		FeeToPerpEcosystemFund: feeToPerpEF,
		FeeToLiquidator:        feeToLiquidator,
		PositionResp:           positionResp,
		BadDebt:                positionResp.BadDebt,
	}, nil

}
