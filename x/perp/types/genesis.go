package types

import (
	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:               DefaultParams(),
		ModuleAccountBalance: sdk.NewCoin(common.CollDenom, sdk.ZeroInt()),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
