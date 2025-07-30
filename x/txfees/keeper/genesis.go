package keeper

import (
	"github.com/NibiruChain/nibiru/v2/x/txfees/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the txfees module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	err := k.SetFeeTokens(ctx, genState.Feetokens)
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns the txfees module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	feetoken := k.GetFeeTokens(ctx)
	genesis.Feetokens = feetoken
	return genesis
}
