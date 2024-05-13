// Copyright (c) 2023-2024 Nibi, Inc.
package evmmodule

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
)

// InitGenesis initializes genesis state based on exported genesis
func InitGenesis(
	ctx sdk.Context,
	k *keeper.Keeper,
	accountKeeper evm.AccountKeeper,
	data evm.GenesisState,
) []abci.ValidatorUpdate {
	// TODO: impl InitGenesis
	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the EVM module
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper, ak evm.AccountKeeper) *evm.GenesisState {
	// TODO: impl ExportGenesis
	return &evm.GenesisState{
		Accounts: []evm.GenesisAccount{},
		Params:   evm.Params{},
	}
}
