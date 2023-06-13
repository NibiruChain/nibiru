package spot

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/spot/keeper"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

// InitGenesis initializes the spot module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.SetNextPoolNumber(ctx, uint64(genState.Params.StartingPoolNumber))

	for _, pool := range genState.Pools {
		k.SetPool(ctx, pool)
	}
}

// ExportGenesis returns the spot module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.Pools = k.FetchAllPools(ctx)

	return genesis
}
