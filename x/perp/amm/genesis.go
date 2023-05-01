package amm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	for _, vp := range genState.Markets {
		if err := k.CreatePool(
			/* ctx */ ctx,
			/* pair */ vp.Pair,
			/* quoteReserve */ vp.QuoteReserve,
			/* baseReserve */ vp.BaseReserve,
			/* config */ vp.Config,
			/* pegMultiplier */ vp.PegMultiplier,
		); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Markets: k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values(),
	}
}
