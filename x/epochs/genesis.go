package epochs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/epochs/keeper"
	"github.com/NibiruChain/nibiru/x/epochs/types"
)

// InitGenesis sets epoch info from genesis
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) (err error) {
	for _, epoch := range genState.Epochs {
		if err = k.AddEpochInfo(ctx, epoch); err != nil {
			return err
		}
	}
	return
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Epochs = k.AllEpochInfos(ctx)

	return genesis
}
