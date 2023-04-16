package inflation

import (
	"github.com/NibiruChain/nibiru/x/inflation/keeper"
	"github.com/NibiruChain/nibiru/x/inflation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	ak types.AccountKeeper,
	_ types.StakingKeeper,
	data types.GenesisState,
) {
	// Ensure inflation module account is set on genesis
	if acc := ak.GetModuleAccount(ctx, types.ModuleName); acc == nil {
		panic("the inflation module account has not been set")
	}

	// Set genesis state
	k.SetParams(ctx, data.Params)

	period := data.Period
	k.CurrentPeriod.Set(ctx, period)

	skippedEpochs := data.SkippedEpochs
	k.NumSkippedEpochs.Set(ctx, skippedEpochs)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:        k.GetParams(ctx),
		Period:        k.CurrentPeriod.Peek(ctx),
		SkippedEpochs: k.NumSkippedEpochs.Peek(ctx),
	}
}
