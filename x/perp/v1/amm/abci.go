package amm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v1/amm/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
)

// EndBlocker Called every block to store a snapshot of the perpamm.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, pool := range k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values() {
		snapshot := types.NewReserveSnapshot(
			pool.Pair,
			pool.BaseReserve,
			pool.QuoteReserve,
			pool.PegMultiplier,
			ctx.BlockTime(),
		)
		k.ReserveSnapshots.Insert(ctx, collections.Join(pool.Pair, ctx.BlockTime()), snapshot)

		_ = ctx.EventManager().EmitTypedEvent(&types.ReserveSnapshotSavedEvent{
			Pair:           snapshot.Pair,
			QuoteReserve:   snapshot.QuoteReserve,
			BaseReserve:    snapshot.BaseReserve,
			MarkPrice:      snapshot.QuoteReserve.Quo(snapshot.BaseReserve),
			BlockHeight:    ctx.BlockHeight(),
			BlockTimestamp: ctx.BlockTime(),
		})
	}
	return []abci.ValidatorUpdate{}
}
