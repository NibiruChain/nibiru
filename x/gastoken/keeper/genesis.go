package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the gas_token module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState, accountKeeper types.AccountKeeper) {
	// Validate first to avoid partially-initialized state on bad input.
	if err := genState.Validate(); err != nil {
		panic(fmt.Errorf("gastoken: invalid genesis: %w", err))
	}
	k.Params.Set(ctx, genState.Params)
	err := k.SetFeeTokens(ctx, genState.Feetokens)
	if err != nil {
		panic(fmt.Errorf("gastoken: SetFeeTokens failed: %w", err))
	}

	if gasTokenModule := accountKeeper.GetModuleAccount(ctx, types.ModuleName); gasTokenModule == nil {
		panic("the GasToken module account has not been set")
	}
}

// ExportGenesis returns the gas_token module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	feetokens := k.GetFeeTokens(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	return &types.GenesisState{
		Params:    params,
		Feetokens: feetokens,
	}
}
