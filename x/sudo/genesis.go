package sudo

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/sudo/types"
)

// InitGenesis initializes the module's state from a provided genesis state JSON.
func InitGenesis(ctx sdk.Context, k Keeper, genState types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}
	k.Sudoers.Set(ctx, genState.Sudoers)
}

// ExportGenesis returns the module's exported genesis state.
// This fn assumes InitGenesis has already been called.
func ExportGenesis(ctx sdk.Context, k Keeper) *types.GenesisState {
	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Sudoers: pbSudoers,
	}
}

func DefaultGenesis() *types.GenesisState {
	return &types.GenesisState{
		Sudoers: types.Sudoers{
			Root:      "",
			Contracts: []string{},
		},
	}
}
