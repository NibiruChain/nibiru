package vpool

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections"

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
		// TODO snapshot.TimestampMs can just be time...
		k.ReserveSnapshots.Insert(ctx, collections.Join(snapshot.Pair, time.UnixMilli(snapshot.TimestampMs)), snapshot)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Vpools:    k.Pools.Iterate(ctx, collections.Range[common.AssetPair]{}).Values(),
		Snapshots: k.ReserveSnapshots.Iterate(ctx, collections.PairRange[common.AssetPair, time.Time]{}).Values(),
	}
}
