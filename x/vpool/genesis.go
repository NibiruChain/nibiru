package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/common"

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
			vp.MaxLeverage,
		)
	}

	for _, snapshot := range genState.Snapshots {
		k.ReserveSnapshots.Insert(ctx, keys.Join(snapshot.Pair, keys.Uint64(uint64(snapshot.TimestampMs))), snapshot)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Vpools:    k.Pools.Iterate(ctx, keys.NewRange[common.AssetPair]()).Values(),
		Snapshots: k.ReserveSnapshots.Iterate(ctx, keys.NewRange[keys.Pair[common.AssetPair, keys.Uint64Key]]()).Values(),
	}
}
