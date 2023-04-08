package keeper

import (
	"fmt"

	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
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

	/* LatestCumulativePremiumFraction: latest cumulative funding rate from state. Units are (margin units)/(position units). */
	LatestCumulativePremiumFraction sdk.Dec
}

func (r RemainingMarginWithFundingPayment) String() string {
	return fmt.Sprintf(
		"RemainingMarginWithFundingPayment{Margin: %s, FundingPayment: %s, PrepaidBadDebt: %s, LatestCumulativePremiumFraction: %s}",
		r.Margin, r.FundingPayment, r.BadDebt, r.LatestCumulativePremiumFraction,
	)
}

func (k Keeper) CalcRemainMarginWithFundingPayment(
	ctx sdk.Context,
	currentPosition types.Position,
	marginDelta sdk.Dec,
) (remaining RemainingMarginWithFundingPayment, err error) {
	remaining.LatestCumulativePremiumFraction, err = k.
		getLatestCumulativePremiumFraction(ctx, currentPosition.Pair)
	if err != nil {
		return remaining, err
	}

	if currentPosition.Size_.IsZero() {
		remaining.FundingPayment = sdk.ZeroDec()
	} else {
		remaining.FundingPayment = (remaining.LatestCumulativePremiumFraction.
			Sub(currentPosition.LatestCumulativePremiumFraction)).
			Mul(currentPosition.Size_)
	}

	remainingMargin := currentPosition.Margin.Add(marginDelta).Sub(remaining.FundingPayment)

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
	ctx sdk.Context, market perpammtypes.Market, pos types.Position,
) (freeCollateral sdk.Dec, err error) {
	if err = pos.Pair.Validate(); err != nil {
		return
	}

	positionNotional, unrealizedPnL, err := k.
		GetPreferencePositionNotionalAndUnrealizedPnL(
			ctx,
			market,
			pos,
			types.PnLPreferenceOption_MIN,
		)
	if err != nil {
		return
	}
	remainingMargin := sdk.MinDec(pos.Margin, pos.Margin.Add(unrealizedPnL))

	maintenanceMarginRatio, err := k.PerpAmmKeeper.GetMaintenanceMarginRatio(ctx, pos.Pair)
	if err != nil {
		return
	}
	maintenanceMarginRequirement := positionNotional.Mul(maintenanceMarginRatio)

	return remainingMargin.Sub(maintenanceMarginRequirement), nil
}

// getLatestCumulativePremiumFraction returns the last cumulative funding rate recorded for the
// specific pair.
func (k Keeper) getLatestCumulativePremiumFraction(
	ctx sdk.Context, pair asset.Pair,
) (sdk.Dec, error) {
	pairMetadata, err := k.PairsMetadata.Get(ctx, pair)
	if err != nil {
		k.Logger(ctx).Error(
			err.Error(),
			"pair",
			pair,
		)
		return sdk.Dec{}, err
	}
	// this should never fail
	return pairMetadata.LatestCumulativePremiumFraction, nil
}
