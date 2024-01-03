package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CalculateEpochProvisions returns mint provision per epoch
func CalculateEpochMintProvision(
	params Params,
	period uint64,
) sdk.Dec {
	if params.EpochsPerPeriod == 0 || !params.InflationEnabled || period >= params.MaxPeriod {
		return sdk.ZeroDec()
	}

	// truncating to the nearest integer
	x := period

	// Calculate the value of the polynomial at x
	polynomialValue := polynomial(params.PolynomialFactors, sdk.NewDec(int64(x)))

	if polynomialValue.IsNegative() {
		// Just to make sure nothing weird occur
		return sdk.ZeroDec()
	}

	return polynomialValue.Quo(sdk.NewDec(int64(params.EpochsPerPeriod)))
}

// Compute the value of x given the polynomial factors
func polynomial(factors []sdk.Dec, x sdk.Dec) sdk.Dec {
	result := sdk.ZeroDec()
	for i, factor := range factors {
		result = result.Add(factor.Mul(x.Power(uint64(len(factors) - i - 1))))
	}

	// Multiply by 1 million to get the value in unibi
	// 1 unibi = 1e6 nibi and the polynomial was fit on nibi token curve.
	return result.Mul(sdk.NewDec(1_000_000))
}
