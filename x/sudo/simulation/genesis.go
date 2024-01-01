package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/sudo/types"
)

func RandomizedGenState(simState *module.SimulationState) {
	rootAddress := simState.Accounts[simState.Rand.Intn(len(simState.Accounts))].Address

	genState := types.GenesisState{
		Sudoers: types.Sudoers{
			Root:      rootAddress.String(),
			Contracts: []string{},
		},
	}

	bz, err := json.MarshalIndent(&genState, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated x/sudo parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&genState)
}
