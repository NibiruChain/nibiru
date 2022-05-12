package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NOTE hardcoded for now. Need to discuss whether this should be part of the
// Params of x/perp
var initMarginRatio = sdk.MustNewDecFromStr("0.1")

type Remaining struct {
	// margin sdk.Int: amount of quote token (y) backing the position.
	margin sdk.Dec

	/* badDebt sdk.Int: Bad debt (margin units) cleared by the PerpEF during the tx.
	   Bad debt is negative net margin past the liquidation point of a position. */
	badDebt sdk.Dec

	/* fundingPayment sdk.Dec: A funding payment made or received by the trader on
	    the current position. 'fundingPayment' is positive if 'owner' is the sender
		and negative if 'owner' is the receiver of the payment. Its magnitude is
		abs(vSize * fundingRate). Funding payments act to converge the mark price
		(vPrice) and index price (average price on major exchanges). */
	fPayment sdk.Dec

	/* latestCPF: latest cumulative premium fraction */
	latestCPF sdk.Dec
}

// TODO test: CalcRemainMarginWithFundingPayment | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) CalcRemainMarginWithFundingPayment(
	ctx sdk.Context, pair common.TokenPair,
	oldPosition *types.Position, marginDelta sdk.Dec,
) (remaining Remaining, err error) {
	remaining.latestCPF, err = k.getLatestCumulativePremiumFraction(ctx, pair)
	if err != nil {
		return
	}

	if oldPosition.Size_.IsZero() {
		remaining.fPayment = sdk.ZeroDec()
	} else {
		remaining.fPayment = remaining.latestCPF.
			Sub(oldPosition.LastUpdateCumulativePremiumFraction).
			Mul(oldPosition.Size_)
	}

	signedRemainMargin := marginDelta.Sub(remaining.fPayment).Add(oldPosition.Margin)

	if signedRemainMargin.IsNegative() {
		// the remaining margin is negative, liquidators didn't do their job
		// and we have negative margin that must come out of the ecosystem fund
		remaining.badDebt = signedRemainMargin.Abs()
	} else {
		remaining.badDebt = sdk.ZeroDec()
		remaining.margin = signedRemainMargin.Abs()
	}

	return remaining, err
}

func (k Keeper) calcFreeCollateral(ctx sdk.Context, pos *types.Position, fundingPayment, badDebt sdk.Dec,
) (sdk.Int, error) {
	owner, err := sdk.AccAddressFromBech32(pos.Address)
	if err != nil {
		return sdk.Int{}, err
	}
	pair, err := common.NewTokenPairFromStr(pos.Pair)
	if err != nil {
		return sdk.Int{}, err
	}
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return sdk.Int{}, err
	}

	unrealizedPnL, positionNotional, err := k.
		getPreferencePositionNotionalAndUnrealizedPnL(
			ctx, pair, owner.String(), types.PnLPreferenceOption_MIN)
	if err != nil {
		return sdk.Int{}, err
	}
	freeMargin := pos.Margin.Add(fundingPayment).Sub(badDebt)
	accountValue := unrealizedPnL.Add(freeMargin)
	minCollateral := sdk.MinDec(accountValue, freeMargin)

	// Get margin requirement. This rounds up, so 16.5 margin required -> 17
	var marginRequirement sdk.Int
	if pos.Size_.IsPositive() {
		// if long position, use open notional
		marginRequirement = initMarginRatio.Mul(pos.OpenNotional).RoundInt()
	} else {
		// if short, use current notional
		marginRequirement = initMarginRatio.Mul(positionNotional).RoundInt()
	}
	freeCollateral := minCollateral.Sub(marginRequirement.ToDec()).TruncateInt()
	return freeCollateral, nil
}
