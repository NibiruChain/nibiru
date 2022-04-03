package math

import (
	"fmt"
	"math"

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
//
// panics if yWeight is 0.
func solveConstantProductInvariant(
	xPrior,
	xAfter,
	xWeight,
	yPrior,
	yWeight sdk.Dec,
) (deltaY sdk.Dec) {
	// weightRatio = (xWeight/yWeight)
	weightRatio := xWeight.Quo(yWeight)

	// r = xPrior/xAfter
	r := xPrior.Quo(xAfter)

	// amountY = balanceY * (1 - (y ^ weightRatio))
	yToWeightRatio := sdk.MustNewDecFromStr(
		fmt.Sprintf("%f", math.Pow(r.MustFloat64(), weightRatio.MustFloat64())),
	)
	paranthetical := sdk.OneDec().Sub(yToWeightRatio)
	amountY := yPrior.Mul(paranthetical)
	return amountY
}
