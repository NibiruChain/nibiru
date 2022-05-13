package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// CreatePool creates a pool for a specific pair.
func (k Keeper) CreatePool(
	ctx sdk.Context,
	pair string,
	tradeLimitRatio sdk.Dec, // integer with 6 decimals, 1_000_000 means 1.0
	quoteAssetReserve sdk.Dec,
	baseAssetReserve sdk.Dec,
	fluctuationLimitRatio sdk.Dec,
	maxOracleSpreadRatio sdk.Dec,
) {
	pool := types.NewPool(
		pair,
		tradeLimitRatio,
		quoteAssetReserve,
		baseAssetReserve,
		fluctuationLimitRatio,
		maxOracleSpreadRatio,
	)

	k.savePool(ctx, pool)
	k.saveSnapshot(ctx, common.TokenPair(pool.Pair), 0, pool.QuoteAssetReserve, pool.BaseAssetReserve, ctx.BlockTime(), ctx.BlockHeight())
	k.saveSnapshotCounter(ctx, common.TokenPair(pair), 0)
}

// getPool returns the pool from database
func (k Keeper) getPool(ctx sdk.Context, pair common.TokenPair) (
	*types.Pool, error,
) {
	bz := ctx.KVStore(k.storeKey).Get(types.GetPoolKey(pair))
	if bz == nil {
		return nil, fmt.Errorf("Could not find vpool for pair %s", pair.String())
	}

	var pool types.Pool
	k.codec.MustUnmarshal(bz, &pool)
	return &pool, nil
}

func (k Keeper) savePool(
	ctx sdk.Context,
	pool *types.Pool,
) {
	bz := k.codec.MustMarshal(pool)
	ctx.KVStore(k.storeKey).Set(types.GetPoolKey(common.TokenPair(pool.Pair)), bz)
}

// ExistsPool returns true if pool exists, false if not.
func (k Keeper) ExistsPool(ctx sdk.Context, pair common.TokenPair) bool {
	return ctx.KVStore(k.storeKey).Has(types.GetPoolKey(pair))
}
