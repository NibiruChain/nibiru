package mintmod

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/mint"
	"github.com/NibiruChain/nibiru/v2/x/mint/keeper"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	ak mint.AccountKeeper,
	_ mint.StakingKeeper,
	data mint.GenesisState,
) {
	// Ensure inflation module account is set on genesis
	if acc := ak.GetModuleAccount(ctx, mint.ModuleName); acc == nil {
		panic("the inflation module account has not been set")
	}

	// Set genesis state
	k.Params.Set(ctx, data.Params)

	period := data.Period
	k.CurrentPeriod.Set(ctx, period)

	skippedEpochs := data.SkippedEpochs
	k.NumSkippedEpochs.Set(ctx, skippedEpochs)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *mint.GenesisState {
	return &mint.GenesisState{
		Params:        k.GetParams(ctx),
		Period:        k.CurrentPeriod.Peek(ctx),
		SkippedEpochs: k.NumSkippedEpochs.Peek(ctx),
	}
}
