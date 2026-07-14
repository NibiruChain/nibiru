package keeper

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis/types"
)

// new crisis genesis
func (k *Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) {
	if err := k.SetConstantFee(ctx, data.ConstantFee); err != nil {
		panic(err)
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	constantFee := k.GetConstantFee(ctx)
	return types.NewGenesisState(constantFee)
}
