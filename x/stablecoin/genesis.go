package stablecoin

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/stablecoin/keeper"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if genState.ModuleAccountBalance.Amount.GT(sdk.ZeroInt()) {
		if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(genState.ModuleAccountBalance)); err != nil {
			panic(err)
		}
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
