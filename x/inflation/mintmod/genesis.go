package mintmod

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/inflation"
	"github.com/NibiruChain/nibiru/v2/x/inflation/keeper"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	ak inflation.AccountKeeper,
	_ inflation.StakingKeeper,
	data inflation.GenesisState,
) {
	// Ensure inflation module account is set on genesis
	if acc := ak.GetModuleAccount(ctx, inflation.ModuleName); acc == nil {
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
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *inflation.GenesisState {
	return &inflation.GenesisState{
		Params:        k.GetParams(ctx),
		Period:        k.CurrentPeriod.Peek(ctx),
		SkippedEpochs: k.NumSkippedEpochs.Peek(ctx),
	}
}
