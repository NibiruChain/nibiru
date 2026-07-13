package simulation

import (
	"math/rand"

	simtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/simulation"

	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/types"
)

// GenChannelGenesis returns the default channel genesis state.
func GenChannelGenesis(_ *rand.Rand, _ []simtypes.Account) types.GenesisState {
	return types.DefaultGenesisState()
}
