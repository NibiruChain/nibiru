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
				sdk.MustNewDecFromStr("-0.00014903"),
				sdk.MustNewDecFromStr("0.07527647"),
				sdk.MustNewDecFromStr("-19.11742154"),
				sdk.MustNewDecFromStr("3170.0969905"),
				sdk.MustNewDecFromStr("-339271.31060432"),
				sdk.MustNewDecFromStr("18063678.8582418"),
			},
			InflationDistribution: types.InflationDistribution{
				CommunityPool:     sdk.NewDecWithPrec(35_159141, 8), // 35.159141%
				StakingRewards:    sdk.NewDecWithPrec(27_757217, 8), // 27.757217%
				StrategicReserves: sdk.NewDecWithPrec(37_083642, 8), // 37.083642%
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
