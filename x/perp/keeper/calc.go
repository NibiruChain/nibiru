package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var initMarginRatio = sdk.MustNewDecFromStr("0.1")

// TODO test: CalcRemainMarginWithFundingPayment | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) CalcRemainMarginWithFundingPayment(
	ctx sdk.Context, pair common.TokenPair,
	oldPosition *types.Position, marginDelta sdk.Int,
) (remaining Remaining, err error) {
	remaining.latestCPF, err = k.GetLatestCumulativePremiumFraction(ctx, pair)
	if err != nil {
		return
	}

	if oldPosition.Size_.IsZero() {
		remaining.fPayment = remaining.latestCPF.
			Sub(oldPosition.LastUpdateCumulativePremiumFraction).
			Mul(oldPosition.Size_)
	} else {
		remaining.fPayment = sdk.ZeroInt()
	}

	signedRemainMargin := marginDelta.Sub(remaining.fPayment).Add(oldPosition.Margin)

	if signedRemainMargin.IsNegative() {
		// the remaining margin is negative, liquidators didn't do their job
		// and we have negative margin that must come out of the ecosystem fund
		remaining.badDebt = signedRemainMargin.Abs()
	} else {
		remaining.badDebt = sdk.ZeroInt()
		remaining.margin = signedRemainMargin.Abs()
	}

	return remaining, err
}

func (k Keeper) calcFreeCollateral(ctx sdk.Context, pos *types.Position, fundingPayment, badDebt sdk.Int,
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
	minCollateral := sdk.MinInt(accountValue, freeMargin)

	// Get margin requirement. This rounds up, so 16.5 margin required -> 17
	var marginRequirement sdk.Int
	if pos.Size_.IsPositive() {
		// if long position, use open notional
		marginRequirement = initMarginRatio.MulInt(pos.OpenNotional).RoundInt()
	} else {
		// if short, use current notional
		marginRequirement = initMarginRatio.MulInt(positionNotional).RoundInt()
	}
	freeCollateral := minCollateral.Sub(marginRequirement)
	return freeCollateral, nil
}
