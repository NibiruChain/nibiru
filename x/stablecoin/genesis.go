package stablecoin

import (
	"github.com/MatrixDao/matrix/x/stablecoin/keeper"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init

	if genState.ModuleAccountBalance.Amount.GT(sdk.ZeroInt()) {
		k.SetCollBalance(ctx, genState.ModuleAccountBalance)
	}
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.ModuleAccountBalance = k.GetModuleAccountBalance(ctx)

	return genesis
}
