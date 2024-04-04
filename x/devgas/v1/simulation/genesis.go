package simulation

// DONTCOVER

import (
	sdkmath "cosmossdk.io/math"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

const (
	DeveloperFeeShare = "developer_fee_share"
)

func GenDeveloperFeeShare(r *rand.Rand) sdkmath.LegacyDec {
	return sdkmath.LegacyNewDecWithPrec(int64(r.Intn(100)), 2)
}

func RandomizedGenState(simState *module.SimulationState) {
	var developerFeeShare sdkmath.LegacyDec
	simState.AppParams.GetOrGenerate(
		DeveloperFeeShare, &developerFeeShare, simState.Rand,
		func(r *rand.Rand) { developerFeeShare = GenDeveloperFeeShare(r) },
	)

	devgasGenesis := types.GenesisState{
		Params: types.ModuleParams{
			EnableFeeShare:  true,
			DeveloperShares: developerFeeShare,
			AllowedDenoms:   []string{},
		},
		FeeShare: []types.FeeShare{},
	}

	bz, err := json.MarshalIndent(&devgasGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated x/devgas parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&devgasGenesis)
}
