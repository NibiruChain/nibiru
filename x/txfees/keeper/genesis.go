package keeper

import (
	"github.com/NibiruChain/nibiru/v2/x/txfees/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the txfees module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	err := k.SetFeeToken(ctx, genState.Feetoken)
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns the txfees module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	feetoken, err := k.GetFeeToken(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Feetoken = feetoken
	return genesis
}
