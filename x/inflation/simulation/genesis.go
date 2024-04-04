package simulation

// DONTCOVER

import (
	sdkmath "cosmossdk.io/math"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// RandomizedGenState generates a random GenesisState for distribution
func RandomizedGenState(simState *module.SimulationState) {
	inflationGenesis := types.GenesisState{
		Params: types.Params{
			InflationEnabled: true,
			PolynomialFactors: []sdkmath.LegacyDec{
				sdkmath.LegacyMustNewDecFromStr("-0.00014851"),
				sdkmath.LegacyMustNewDecFromStr("0.07501029"),
				sdkmath.LegacyMustNewDecFromStr("-19.04983993"),
				sdkmath.LegacyMustNewDecFromStr("3158.89198346"),
				sdkmath.LegacyMustNewDecFromStr("-338072.17402939"),
				sdkmath.LegacyMustNewDecFromStr("17999834.20786474"),
			},
			InflationDistribution: types.InflationDistribution{
				CommunityPool:     sdkmath.LegacyNewDecWithPrec(35_142714, 8), // 35.142714%
				StakingRewards:    sdkmath.LegacyNewDecWithPrec(27_855672, 8), // 27.855672%
				StrategicReserves: sdkmath.LegacyNewDecWithPrec(37_001614, 8), // 37.001614%
			},
			EpochsPerPeriod: 30,
			PeriodsPerYear:  12,
			MaxPeriod:       8 * 12,
		},
		Period:        0,
		SkippedEpochs: 0,
	}

	bz, err := json.MarshalIndent(&inflationGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated x/inflation parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&inflationGenesis)
}
