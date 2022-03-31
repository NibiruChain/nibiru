package lockup

import (
	"github.com/MatrixDao/matrix/x/lockup/keeper"
	"github.com/MatrixDao/matrix/x/lockup/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.LockupKeeper, genState types.GenesisState) {

}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.LockupKeeper) *types.GenesisState {
	return &types.GenesisState{}
}
