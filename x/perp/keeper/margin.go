package keeper

import (
	"fmt"

	v1 "github.com/NibiruChain/nibiru/x/perp/types/v1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO test: GetMarginRatio
func (k Keeper) GetMarginRatio(ctx sdk.Context, amm v1.IVirtualPool, trader string) (sdk.Int, error) {
	position, err := k.Positions().Get(ctx, amm.Pair(), trader) // TODO(mercilex): inefficient position get
	if err != nil {
		return sdk.Int{}, err
	}

	if position.Size_.IsZero() {
		panic("position with zero size") // tODO(mercilex): panic or error? this is a require
	}

	unrealizedPnL, positionNotional, err := k.getPreferencePositionNotionalAndUnrealizedPnL(
		ctx, amm, trader, v1.PnLPreferenceOption_PnLPreferenceOption_MAX)
	if err != nil {
		return sdk.Int{}, err
	}

	remainMargin, badDebt, _, _, err := k.calcRemainMarginWithFundingPayment(
		ctx, amm.Pair(), position, unrealizedPnL)
	if err != nil {
		return sdk.Int{}, err
	}

	return remainMargin.Sub(badDebt).Quo(positionNotional), nil
}

/*
function requireMoreMarginRatio(
        SignedDecimal.signedDecimal memory _marginRatio,
        Decimal.decimal memory _baseMarginRatio,
        bool _largerThanOrEqualTo
    ) private pure {
        int256 remainingMarginRatio = _marginRatio.subD(_baseMarginRatio).toInt();
        require(
            _largerThanOrEqualTo ? remainingMarginRatio >= 0 : remainingMarginRatio < 0,
            "Margin ratio not meet criteria"
        );
    }
*/

// TODO test: requireMoreMarginRatio
func requireMoreMarginRatio(marginRatio, baseMarginRatio sdk.Int, largerThanOrEqualTo bool) error {
	// TODO(mercilex): look at this and make sure it's legit compared ot the counterparty above ^
	remainMarginRatio := marginRatio.Sub(baseMarginRatio)
	switch largerThanOrEqualTo {
	case true:
		if !remainMarginRatio.GTE(sdk.ZeroInt()) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	default:
		if remainMarginRatio.LT(sdk.ZeroInt()) {
			return fmt.Errorf("margin ratio did not meet criteria")
		}
	}

	return nil
}

// TODO test: calcRemainMarginWithFundingPayment | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) calcRemainMarginWithFundingPayment(
	ctx sdk.Context, vpool string,
	oldPosition *v1.Position, marginDelta sdk.Int,
) (remainMargin sdk.Int, badDebt sdk.Int, fundingPayment sdk.Int,
	latestCumulativePremiumFraction sdk.Int, err error) {
	latestCumulativePremiumFraction, err = k.GetLatestCumulativePremiumFraction(ctx, vpool)
	if err != nil {
		return
	}

	if !oldPosition.Size_.IsZero() { // TODO(mercilex): what if this does evaluate to false?
		fundingPayment = latestCumulativePremiumFraction.
			Sub(oldPosition.LastUpdateCumulativePremiumFraction).
			Mul(oldPosition.Size_)
	}

	signedRemainMargin := marginDelta.Sub(fundingPayment).Add(oldPosition.Margin)
	switch signedRemainMargin.IsNegative() {
	case true:
		badDebt = signedRemainMargin.Abs()
	case false:
		badDebt = sdk.ZeroInt()
		remainMargin = signedRemainMargin.Abs()
	}

	return
}
