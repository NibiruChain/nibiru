package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CalculateEpochProvisions returns mint provision per epoch
func CalculateEpochMintProvision(
	params Params,
	period uint64,
) sdk.Dec {
	if params.EpochsPerPeriod == 0 {
		return sdk.ZeroDec()
	}

	x := period                          // period
	a := params.ExponentialCalculation.A // initial value
	r := params.ExponentialCalculation.R // reduction factor
	c := params.ExponentialCalculation.C // long term inflation

	// exponentialDecay := a * (1 - r) ^ x + c
	decay := sdk.OneDec().Sub(r)
	periodProvision := a.Mul(decay.Power(x)).Add(c)

	// epochProvision = periodProvision / epochsPerPeriod
	epochProvision := periodProvision.QuoInt64(int64(params.EpochsPerPeriod))

	// Multiply epochMintProvision with power reduction (10^6 for unibi) as the
	// calculation is based on `NIBI` and the issued tokens need to be given in
	// `uNIBI`
	epochProvision = epochProvision.MulInt(sdk.DefaultPowerReduction)
	return epochProvision
}
