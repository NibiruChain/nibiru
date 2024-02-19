package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// RandomizedGenState generates a random GenesisState for distribution
func RandomizedGenState(simState *module.SimulationState) {
	inflationGenesis := types.GenesisState{
		Params: types.Params{
			InflationEnabled: true,
			PolynomialFactors: []sdk.Dec{
				sdk.MustNewDecFromStr("-0.00014851"),
				sdk.MustNewDecFromStr("0.07501001"),
				sdk.MustNewDecFromStr("-19.04980404"),
				sdk.MustNewDecFromStr("3158.89014745"),
				sdk.MustNewDecFromStr("-338072.13773281"),
				sdk.MustNewDecFromStr("17999834.05992003"),
			},
			InflationDistribution: types.InflationDistribution{
				CommunityPool:     sdk.NewDecWithPrec(35_142714, 8), // 35.142714%
				StakingRewards:    sdk.NewDecWithPrec(27_855672, 8), // 27.855672%
				StrategicReserves: sdk.NewDecWithPrec(37_001614, 8), // 37.001614%
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
