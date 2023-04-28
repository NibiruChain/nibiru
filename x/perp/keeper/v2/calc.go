package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type RemainingMarginWithFundingPayment struct {
	// MarginAbs: amount of quote token (y) backing the position.
	MarginAbs sdk.Dec

	/* BadDebtAbs: Bad debt (margin units) cleared by the PerpEF during the tx.
	   Bad debt is negative net margin past the liquidation point of a position. */
	BadDebtAbs sdk.Dec

	/* FundingPayment: A funding payment (margin units) made or received by the trader on
	    the current position. 'fundingPayment' is positive if 'owner' is the sender
		and negative if 'owner' is the receiver of the payment. Its magnitude is
		abs(vSize * fundingRate). Funding payments act to converge the mark price
		(vPrice) and index price (average price on major exchanges).
	*/
	FundingPayment sdk.Dec
}

func (r RemainingMarginWithFundingPayment) String() string {
	return fmt.Sprintf(
		"RemainingMarginWithFundingPayment{Margin: %s, FundingPayment: %s, PrepaidBadDebt: %s}",
		r.MarginAbs, r.FundingPayment, r.BadDebtAbs,
	)
}

// CalcRemainMarginWithFundingPayment calculates the remaining margin after a margin delta is applied.
func CalcRemainMarginWithFundingPayment(
	currentPosition v2types.Position,
	marginDeltaSigned sdk.Dec,
	market v2types.Market,
) (remaining RemainingMarginWithFundingPayment, err error) {
	if currentPosition.Size_.IsZero() {
		remaining.FundingPayment = sdk.ZeroDec()
	} else {
		remaining.FundingPayment = (market.LatestCumulativePremiumFraction.
			Sub(currentPosition.LatestCumulativePremiumFraction)).
			Mul(currentPosition.Size_)
	}

	remainingMargin := currentPosition.Margin.Add(marginDeltaSigned).Sub(remaining.FundingPayment)

	if remainingMargin.IsNegative() {
		// the remaining margin is negative, liquidators didn't do their job
		// and we have negative margin that must come out of the ecosystem fund
		remaining.BadDebtAbs = remainingMargin.Abs()
		remaining.MarginAbs = sdk.ZeroDec()
	} else {
		remaining.MarginAbs = remainingMargin.Abs()
		remaining.BadDebtAbs = sdk.ZeroDec()
	}

	return remaining, err
}

/*
	calcFreeCollateral computes the amount of collateral backing the position that can

be removed without giving the position bad debt. Factors in the unrealized PnL when
calculating free collateral.

Args:
- ctx: Carries information about the current state of the SDK application.
- pos: position for which to compute free collateral.

Returns:
- freeCollateral: Amount of collateral (margin) that can be removed from the
position without making it go underwater.
- err: error
*/
func (k Keeper) calcFreeCollateral(
	ctx sdk.Context, market v2types.Market, amm v2types.AMM, position v2types.Position,
) (freeCollateralSigned sdk.Dec, err error) {
	spotNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return sdk.Dec{}, err
	}
	twapNotional, err := k.PositionNotionalTWAP(ctx, market, amm, position)
	if err != nil {
		return sdk.Dec{}, err
	}
	positionNotional := sdk.MinDec(spotNotional, twapNotional)
	unrealizedPnlSigned := UnrealizedPnl(position, positionNotional)

	maintenanceMarginRequirementAbs := positionNotional.Mul(market.MaintenanceMarginRatio)

	// account for negative unrealizedPnl
	remainingMarginSigned := sdk.MinDec(position.Margin, position.Margin.Add(unrealizedPnlSigned))

	return remainingMarginSigned.Sub(maintenanceMarginRequirementAbs), nil
}
