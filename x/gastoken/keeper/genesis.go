package keeper

import (
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the gas_token module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState, accountKeeper types.AccountKeeper) {
	k.Params.Set(ctx, genState.Params)
	err := k.SetFeeTokens(ctx, genState.Feetokens)
	if err != nil {
		panic(err)
	}

	if gasTokenModule := accountKeeper.GetModuleAccount(ctx, types.ModuleName); gasTokenModule == nil {
		panic("the GasToken module account has not been set")
	}
}

// ExportGenesis returns the gas_token module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	feetoken := k.GetFeeTokens(ctx)
	params, _ := k.GetParams(ctx)
	genesis.Feetokens = feetoken
	genesis.Params = params
	return genesis
}
