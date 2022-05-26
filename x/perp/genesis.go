package perp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if genState.ModuleAccountBalance.Amount.GT(sdk.ZeroInt()) {
		if err := k.BankKeeper.MintCoins(
			ctx, types.ModuleName, sdk.NewCoins(genState.ModuleAccountBalance),
		); err != nil {
			panic(err)
		}
	}

	k.SetParams(ctx, genState.Params)

	for _, pm := range genState.PairMetadata {
		k.PairMetadata().Set(ctx, pm)
	}

	// See https://github.com/cosmos/cosmos-sdk/issues/5569 on why we do this.
	k.AccountKeeper.GetModuleAccount(ctx, types.FeePoolModuleAccount)
	k.AccountKeeper.GetModuleAccount(ctx, types.VaultModuleAccount)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	genesis.Params = k.GetParams(ctx)
	genesis.ModuleAccountBalance = k.GetModuleAccountBalance(ctx, common.GovDenom)
	genesis.PairMetadata = k.PairMetadata().GetAll(ctx)

	return genesis
}
