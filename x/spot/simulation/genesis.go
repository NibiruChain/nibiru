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
	spotGenesis := types.DefaultGenesis()

	bz, err := json.MarshalIndent(&spotGenesis, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected randomly generated x/spot parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(spotGenesis)
}
