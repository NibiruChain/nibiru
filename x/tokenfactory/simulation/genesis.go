package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

const (
	DenomCreationGasConsume = "denom_creation_gas_consume"
)

func GenDenomCreationGasConsume(r *rand.Rand) uint64 {
	return uint64(r.Intn(4e6))
}

// RandomizedGenState generates a random GenesisState for distribution
func RandomizedGenState(simState *module.SimulationState) {
	var denomCreationGasConsume uint64
	simState.AppParams.GetOrGenerate(
		DenomCreationGasConsume, &denomCreationGasConsume, simState.Rand,
		func(r *rand.Rand) { denomCreationGasConsume = GenDenomCreationGasConsume(r) },
	)

	tokenfactoryGenesis := types.GenesisState{
		Params: types.ModuleParams{
			DenomCreationGasConsume: denomCreationGasConsume,
		},
		FactoryDenoms: []types.GenesisDenom{},
	}

	bz, err := json.MarshalIndent(&tokenfactoryGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated x/tokenfactory parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&tokenfactoryGenesis)
}
