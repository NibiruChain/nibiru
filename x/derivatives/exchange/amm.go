package exchange

import (
	"context"
	vammv1 "github.com/MatrixDao/matrix/api/vamm"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func swapInput(
	ctx context.Context,
	amm VirtualAMM, side vammv1.Direction,
	input sdk.Int, minOutput sdk.Int,
	overFluctuateLimit bool,
) (outputAmount sdk.Int, err error) {

	resp, err := amm.SwapInput(ctx, &vammv1.SwapInputRequest{
		Direction:               side,
		QuoteAssetAmount:        input.String(),     // TODO(mercilex): inefficient back and forth conversion
		BaseAssetAmountLimit:    minOutput.String(), // TODO(mercilex): inefficient back and forth conversion
		CanOverFluctuationLimit: overFluctuateLimit,
	})

	if err != nil {
		return sdk.Int{}, err
	}

	outputAmount, ok := sdk.NewIntFromString(resp.Resp) // TODO(mercilex): inefficient back and forth conversion
	if !ok {
		panic("AMM returned invalid output string: " + resp.Resp)
	}

	switch side {
	case vammv1.Direction_DIRECTION_LONG:
		return outputAmount, nil
	default:
		// inverse sign
		return outputAmount.MulRaw(-1), nil
	}
}
