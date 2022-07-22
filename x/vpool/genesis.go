package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/vpool/keeper"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	for _, vp := range genState.Vpools {
		k.CreatePool(
			ctx,
			vp.Pair,
			vp.TradeLimitRatio,
			vp.QuoteAssetReserve,
			vp.BaseAssetReserve,
			vp.FluctuationLimitRatio,
			vp.MaxOracleSpreadRatio,
			vp.MaintenanceMarginRatio,
		)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	pools := k.GetAllPools(ctx)

	var genState types.GenesisState
	genState.Vpools = pools

	return &genState
}
