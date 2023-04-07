package vpool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

// EndBlocker Called every block to store a snapshot of the vpool.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, pool := range k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values() {
		snapshot := types.NewReserveSnapshot(
			pool.Pair,
			pool.BaseAssetReserve,
			pool.QuoteAssetReserve,
			ctx.BlockTime(),
		)
		k.ReserveSnapshots.Insert(ctx, collections.Join(pool.Pair, ctx.BlockTime()), snapshot)

		_ = ctx.EventManager().EmitTypedEvent(&types.ReserveSnapshotSavedEvent{
			Pair:           snapshot.Pair,
			QuoteReserve:   snapshot.QuoteAssetReserve,
			BaseReserve:    snapshot.BaseAssetReserve,
			MarkPrice:      snapshot.QuoteAssetReserve.Quo(snapshot.BaseAssetReserve),
			BlockHeight:    ctx.BlockHeight(),
			BlockTimestamp: ctx.BlockTime(),
		})
	}
	return []abci.ValidatorUpdate{}
}
