package devgas

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/devgas/v1/keeper"
	"github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	data types.GenesisState,
) {
	if err := data.Validate(); err != nil {
		panic(err)
	}

	k.ModuleParams.Set(ctx, data.Params.Sanitize())

	for _, share := range data.FeeShare {
		// Set initial contracts receiving transaction fees
		k.SetFeeShare(ctx, share)
	}
}

// ExportGenesis export module state
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:   k.GetParams(ctx),
		FeeShare: k.DevGasStore.Iterate(ctx, collections.Range[string]{}).Values(),
	}
}
