package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/keeper"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	for _, vp := range genState.Vpools {
		if err := k.CreatePool(
			ctx,
			vp.Pair,
			vp.QuoteAssetReserve,
			vp.BaseAssetReserve,
			vp.Config,
		); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Vpools: k.Pools.Iterate(ctx, collections.Range[common.AssetPair]{}).Values(),
	}
}
