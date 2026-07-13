package simulation

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz"
)

// RandomizedGenState generates a random GenesisState for authz.
func RandomizedGenState(simState *module.SimulationState) {
	simState.GenState[authz.ModuleName] = simState.Cdc.MustMarshalJSON(authz.DefaultGenesisState())
}
