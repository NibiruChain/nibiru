package exchange

import (
	"context"
	"fmt"
	vammv1 "github.com/MatrixDao/matrix/api/vamm"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PnLCalcOption uint8

const (
	PnLCalcOptionTWAP = iota
	PnLCalcOptionSpot
)

func (e Exchange) getPositionNotionalAndUnrealizedPnL(ctx context.Context, amm VirtualAMM, position *parsedPosition, pnlCalcOption PnLCalcOption) (
	positionNotional, unrealizedPnL sdk.Int, err error,
) {
	if position.size.IsZero() {
		return // TODO(mercilex): idk what this return means
	}

	// assess trade direction
	var direction vammv1.Direction
	switch position.size.IsNegative() {
	case true:
		direction = vammv1.Direction_DIRECTION_SHORT
	case false:
		direction = vammv1.Direction_DIRECTION_LONG
	}

	// calculate position notional
	switch pnlCalcOption {
	case PnLCalcOptionTWAP:
		resp, err := amm.GetOutputTWAP(ctx, &vammv1.GetOutputTWAPRequest{
			Direction:       direction,
			BaseAssetAmount: "",
		})
	case PnLCalcOptionSpot:
	default:
		panic("unrecognized pnl calculation option")
	}

	// calculate unrealized PnL

	// gg
	return
}

func (e Exchange) calculateRemainingMarginWithFundingPayment(ctx context.Context, amm VirtualAMM, oldPosition *parsedPosition, marginDelta sdk.Int) (
	remainingMargin, badDebt, fundingPayment, latestCumulativePremiumFraction sdk.Int, err error,
) {
	latestCumulativePremiumFraction, err = e.getLatestCumulativePremiumFraction(ctx, amm)
	if err != nil {
		return
	}

	if !oldPosition.size.IsZero() {
		fundingPayment = latestCumulativePremiumFraction.
			Sub(oldPosition.lastUpdatedCumulativePremiumFraction).
			Mul(oldPosition.size)
	}

	signedRemainMargin := marginDelta.Sub(fundingPayment).Add(oldPosition.margin)

	switch signedRemainMargin.IsNegative() {
	case true:
		badDebt = signedRemainMargin.Abs()
	case false:
		remainingMargin = signedRemainMargin.Abs()
	}

	return
}

func (e Exchange) getLatestCumulativePremiumFraction(ctx context.Context, amm VirtualAMM) (sdk.Int, error) {
	info, err := e.state.VirtualAMMInfoTable().Get(ctx, amm.Pair())
	if err != nil {
		panic(err) // bad initialization
	}

	if len(info.CumulativePremiumFractions) == 0 {
		panic("empty cumulative premium fractions")
	}

	// get latest
	latestString := info.CumulativePremiumFractions[len(info.CumulativePremiumFractions)-1]
	latest, ok := sdk.NewIntFromString(latestString)
	if !ok {
		panic(fmt.Errorf("invalid cumulative premium fraction: %s", latestString))
	}

	return latest, nil
}
