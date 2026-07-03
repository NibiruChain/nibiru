package simulation

import (
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// RandomizedGenState generates a random GenesisState for authz.
func RandomizedGenState(simState *module.SimulationState) {
	simState.GenState[authz.ModuleName] = simState.Cdc.MustMarshalJSON(authz.DefaultGenesisState())
}
