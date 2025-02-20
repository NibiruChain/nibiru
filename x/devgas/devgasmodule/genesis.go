package devgasmodule

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/x/devgas"
	"github.com/NibiruChain/nibiru/v2/x/devgas/keeper"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	data devgas.GenesisState,
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
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *devgas.GenesisState {
	return &devgas.GenesisState{
		Params:   k.GetParams(ctx),
		FeeShare: k.DevGasStore.Iterate(ctx, collections.Range[string]{}).Values(),
	}
}
