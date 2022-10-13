package simulation

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// RandomizedGenState generates a random GenesisState for the perp module
func RandomizedGenState(simState *module.SimulationState) {
	pricefeedGenesis := types.DefaultGenesis()
	for _, acc := range simState.Accounts {
		pricefeedGenesis.GenesisOracles = append(pricefeedGenesis.GenesisOracles, acc.Address.String())
	}

	bz, err := json.MarshalIndent(pricefeedGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated pricefeed state:\n%s\n", bz)

	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(pricefeedGenesis)
}
