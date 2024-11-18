package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

// RandomizedGenState generates a random GenesisState for distribution
func RandomizedGenState(simState *module.SimulationState) {
	inflationGenesis := types.GenesisState{
		Params: types.Params{
			InflationEnabled: true,
			PolynomialFactors: []sdk.Dec{
				math.LegacyMustNewDecFromStr("-0.00014851"),
				math.LegacyMustNewDecFromStr("0.07501029"),
				math.LegacyMustNewDecFromStr("-19.04983993"),
				math.LegacyMustNewDecFromStr("3158.89198346"),
				math.LegacyMustNewDecFromStr("-338072.17402939"),
				math.LegacyMustNewDecFromStr("17999834.20786474"),
			},
			InflationDistribution: types.InflationDistribution{
				CommunityPool:     math.LegacyNewDecWithPrec(35_142714, 8), // 35.142714%
				StakingRewards:    math.LegacyNewDecWithPrec(27_855672, 8), // 27.855672%
				StrategicReserves: math.LegacyNewDecWithPrec(37_001614, 8), // 37.001614%
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
