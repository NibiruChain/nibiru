package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

func (k Keeper) saveOrUpdateSnapshot(
	ctx sdk.Context,
	pair string,
	price sdk.Dec,
) {
	snapshot := &types.PriceSnapshot{
		Price:       price,
		TimestampMs: ctx.BlockTime().UnixMilli(),
	}

	ctx.KVStore(k.storeKey).Set(
		types.PriceSnapshotKey(pair, ctx.BlockHeight()),
		k.cdc.MustMarshal(snapshot),
	)
}

func (k Keeper) IteratePriceSnapshotsFrom(
	ctx sdk.Context,
	start, end []byte,
	reverse bool,
	do func(*types.PriceSnapshot) (stop bool),
) {
	kvStore := ctx.KVStore(k.storeKey)
	iter := kvStore.Iterator(start, end)
	if reverse {
		iter = kvStore.ReverseIterator(start, end)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		priceSnapshot := &types.PriceSnapshot{}
		k.cdc.MustUnmarshal(iter.Value(), priceSnapshot)
		if do(priceSnapshot) {
			break
		}
	}
}
