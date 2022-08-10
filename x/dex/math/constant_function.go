package math

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// solveConstantFunctionInvariant solves the constant function of an AMM
// that determines the relationship between the differences of two sides
// of assets inside the pool.
// For fixed xPrior, xAfter, xWeight, yPrior, yWeight,
// we could deduce the deltaY, calculated by:
// deltaY = balanceY * (1 - (xPrior/xAfter)^(xWeight/yWeight))
// deltaY is positive when y's balance liquidity decreases.
// deltaY is negative when y's balance liquidity increases.
// panics if yWeight is 0.
//
// TODO(https://github.com/NibiruChain/nibiru/issues/141): Currently always calculates the invariant assuming constant weight (xy=k).
// Once we figure out the floating point arithmetic conversions for exponentiation, we can
// add unequal weights.
func SolveConstantProductInvariant(
	xPrior,
	xAfter,
	/*unused*/ _xWeight,
	yPrior,
	/*unused*/ _yWeight sdk.Dec,
) (deltaY sdk.Dec) {
	// // weightRatio = (xWeight/yWeight)
	// weightRatio := xWeight.Quo(yWeight)

	// r = xPrior/xAfter
	r := xPrior.Quo(xAfter)

	// TODO(https://github.com/NibiruChain/nibiru/issues/141): Figure out floating point arithmetic for exponentation.
	// Naive calculation could lead to significant rounding errors with large numbers
	// amountY = yPrior * (1 - (r ^ weightRatio))
	// rToWeightRatio := sdk.MustNewDecFromStr(
	// 	fmt.Sprintf("%f", math.Pow(r.MustFloat64(), weightRatio.MustFloat64())),
	// )
	return yPrior.Mul(sdk.OneDec().Sub(r))
}
