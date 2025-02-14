package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/v2/x/devgas"
)

const (
	DeveloperFeeShare = "developer_fee_share"
)

func GenDeveloperFeeShare(r *rand.Rand) sdk.Dec {
	return math.LegacyNewDecWithPrec(int64(r.Intn(100)), 2)
}

func RandomizedGenState(simState *module.SimulationState) {
	var developerFeeShare sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DeveloperFeeShare, &developerFeeShare, simState.Rand,
		func(r *rand.Rand) { developerFeeShare = GenDeveloperFeeShare(r) },
	)

	devgasGenesis := devgas.GenesisState{
		Params: devgas.ModuleParams{
			EnableFeeShare:  true,
			DeveloperShares: developerFeeShare,
			AllowedDenoms:   []string{},
		},
		FeeShare: []devgas.FeeShare{},
	}

	bz, err := json.MarshalIndent(&devgasGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated x/devgas parameters:\n%s\n", bz)
	simState.GenState[devgas.ModuleName] = simState.Cdc.MustMarshalJSON(&devgasGenesis)
}
