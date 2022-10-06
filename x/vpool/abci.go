package vpool

import (
	"github.com/NibiruChain/nibiru/coll"
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
		k.ReserveSnapshots.Insert(ctx, coll.Join(pool.Pair, ctx.BlockTime()), snapshot)
	}
	return []abci.ValidatorUpdate{}
}
