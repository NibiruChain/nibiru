package simulation

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
)

// RandomizedGenState generates a random GenesisState for the perp module
func RandomizedGenState(simState *module.SimulationState) {
	perpGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		PairMetadata: []types.PairMetadata{
			{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
		},
		Positions:       []types.Position{},
		PrepaidBadDebts: []types.PrepaidBadDebt{},
	}
	perpGenesisBytes, err := json.MarshalIndent(&perpGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated perp genesis:\n%s\n", perpGenesisBytes)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&perpGenesis)
}
