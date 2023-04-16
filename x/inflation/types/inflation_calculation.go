// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/NibiruChain/nibiru/blob/main/LICENSE

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
