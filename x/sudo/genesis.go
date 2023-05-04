package sudo

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/sudo/pb"
)

// InitGenesis initializes the module's state from a provided genesis state JSON.
func InitGenesis(ctx sdk.Context, k Keeper, genState pb.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}
	k.Sudoers.Set(ctx, genState.Sudoers)
}

// ExportGenesis returns the module's exported genesis state.
// This fn assumes InitGenesis has already been called.
func ExportGenesis(ctx sdk.Context, k Keeper) *pb.GenesisState {
	pbSudoers, err := k.Sudoers.Get(ctx)
	if err != nil {
		panic(err)
	}

	return &pb.GenesisState{
		Sudoers: pbSudoers,
	}
}

func DefaultGenesis() *pb.GenesisState {
	return &pb.GenesisState{
		Sudoers: pb.Sudoers{
			Root:      "",
			Contracts: []string{},
		},
	}
}
