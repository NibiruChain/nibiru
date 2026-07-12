package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/v2/x/inflation"
)

// RandomizedGenState generates a random GenesisState for distribution
func RandomizedGenState(simState *module.SimulationState) {
	inflationGenesis := inflation.GenesisState{
		Params: inflation.Params{
			InflationEnabled: true,
			PolynomialFactors: []sdkmath.LegacyDec{
				sdkmath.LegacyMustNewDecFromStr("-0.00014851"),
				sdkmath.LegacyMustNewDecFromStr("0.07501029"),
				sdkmath.LegacyMustNewDecFromStr("-19.04983993"),
				sdkmath.LegacyMustNewDecFromStr("3158.89198346"),
				sdkmath.LegacyMustNewDecFromStr("-338072.17402939"),
				sdkmath.LegacyMustNewDecFromStr("17999834.20786474"),
			},
			InflationDistribution: inflation.InflationDistribution{
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
	simState.GenState[inflation.ModuleName] = simState.Cdc.MustMarshalJSON(&inflationGenesis)
}
