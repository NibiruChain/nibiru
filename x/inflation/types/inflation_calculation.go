package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CalculateEpochMintProvision returns mint provision per epoch
func CalculateEpochMintProvision(
	params Params,
	period uint64,
) sdk.Dec {
	if params.EpochsPerPeriod == 0 || !params.InflationEnabled || period >= params.MaxPeriod {
		return math.LegacyZeroDec()
	}

	// truncating to the nearest integer
	x := period

	// Calculate the value of the polynomial at x
	polynomialValue := polynomial(params.PolynomialFactors, math.LegacyNewDec(int64(x)))

	if polynomialValue.IsNegative() {
		// Just to make sure nothing weird occur
		return math.LegacyZeroDec()
	}

	return polynomialValue.Quo(math.LegacyNewDec(int64(params.EpochsPerPeriod)))
}

// Compute the value of x given the polynomial factors
func polynomial(factors []sdk.Dec, x sdk.Dec) sdk.Dec {
	result := math.LegacyZeroDec()
	for i, factor := range factors {
		result = result.Add(factor.Mul(x.Power(uint64(len(factors) - i - 1))))
	}

	// Multiply by 1 million to get the value in unibi
	// 1 unibi = 1e6 nibi and the polynomial was fit on nibi token curve.
	return result.Mul(math.LegacyNewDec(1_000_000))
}
