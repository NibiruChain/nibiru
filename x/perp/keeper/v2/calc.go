package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type RemainingMarginWithFundingPayment struct {
	// Margin: amount of quote token (y) backing the position.
	Margin sdk.Dec

	/* BadDebt: Bad debt (margin units) cleared by the PerpEF during the tx.
	   Bad debt is negative net margin past the liquidation point of a position. */
	BadDebt sdk.Dec

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
		r.Margin, r.FundingPayment, r.BadDebt,
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
		remaining.BadDebt = remainingMargin.Abs()
		remaining.Margin = sdk.ZeroDec()
	} else {
		remaining.Margin = remainingMargin.Abs()
		remaining.BadDebt = sdk.ZeroDec()
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
	ctx sdk.Context, market v2types.Market, amm v2types.AMM, pos v2types.Position,
) (freeCollateral sdk.Dec, err error) {
	positionNotional, unrealizedPnL, err := k.
		getPositionNotionalAndUnrealizedPnL(
			ctx,
			market,
			amm,
			pos,
			types.PnLCalcOption_SPOT_PRICE,
		)
	if err != nil {
		return
	}
	remainingMargin := sdk.MinDec(pos.Margin, pos.Margin.Add(unrealizedPnL))
	maintenanceMarginRequirement := positionNotional.Mul(market.MaintenanceMarginRatio)
	return remainingMargin.Sub(maintenanceMarginRequirement), nil
}
