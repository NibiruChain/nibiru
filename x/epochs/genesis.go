package epochs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/epochs/keeper"
	"github.com/NibiruChain/nibiru/v2/x/epochs/types"
)

// InitGenesis sets epoch info from genesis
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) (err error) {
	err = genState.Validate()
	if err != nil {
		return
	}
	for _, epoch := range genState.Epochs {
		if err = k.AddEpochInfo(ctx, epoch); err != nil {
			return err
		}
	}
	return
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesisFromTime(ctx.BlockTime())
	genesis.Epochs = k.AllEpochInfos(ctx)

	return genesis
}
