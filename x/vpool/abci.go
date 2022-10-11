package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/vpool/types"

	"github.com/NibiruChain/nibiru/x/vpool/keeper"
)

// EndBlocker Called every block to store a snapshot of the vpool.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, pool := range k.Pools.Iterate(ctx, keys.NewRange[common.AssetPair]()).Values() {
		snapshot := types.NewReserveSnapshot(
			pool.Pair,
			pool.BaseAssetReserve,
			pool.QuoteAssetReserve,
			ctx.BlockTime(),
		)
		k.ReserveSnapshots.Insert(ctx, keys.Join(pool.Pair, keys.Uint64(uint64(ctx.BlockTime().UnixMilli()))), snapshot)

		_ = ctx.EventManager().EmitTypedEvent(&types.ReserveSnapshotSavedEvent{
			Pair:         snapshot.Pair.String(),
			QuoteReserve: snapshot.QuoteAssetReserve,
			BaseReserve:  snapshot.BaseAssetReserve,
		})
	}
	return []abci.ValidatorUpdate{}
}
