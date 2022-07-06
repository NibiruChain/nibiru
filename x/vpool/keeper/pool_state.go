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
	pair common.AssetPair,
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
	k.saveSnapshot(ctx, pair, 0, pool.QuoteAssetReserve, pool.BaseAssetReserve, ctx.BlockTime(), ctx.BlockHeight())
	k.saveSnapshotCounter(ctx, pair, 0)
}

// getPool returns the pool from database
func (k Keeper) getPool(ctx sdk.Context, pair common.AssetPair) (
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
	ctx.KVStore(k.storeKey).Set(types.GetPoolKey(pool.Pair), bz)
}

/*
Saves an updated pool to state and snapshots it.

args:
  - ctx: cosmos-sdk context
  - updatedPool: pool object to save to state
  - skipFluctuationCheck: override fluctuation check from last snapshot

ret:
  - err: error
*/
func (k Keeper) savePoolAndSnapshot(
	ctx sdk.Context,
	updatedPool *types.Pool,
	skipFluctuationCheck bool,
) (err error) {
	// Check if its over Fluctuation Limit Ratio.
	if !skipFluctuationCheck {
		if err = k.checkFluctuationLimitRatio(ctx, updatedPool); err != nil {
			return err
		}
	}

	if err = k.addReserveSnapshot(
		ctx,
		updatedPool.Pair,
		updatedPool.QuoteAssetReserve,
		updatedPool.BaseAssetReserve,
	); err != nil {
		return fmt.Errorf("error creating snapshot: %w", err)
	}

	k.savePool(ctx, updatedPool)

	return nil
}

// ExistsPool returns true if pool exists, false if not.
func (k Keeper) ExistsPool(ctx sdk.Context, pair common.AssetPair) bool {
	return ctx.KVStore(k.storeKey).Has(types.GetPoolKey(pair))
}

// GetAllPools returns all pools that exist.
func (k Keeper) GetAllPools(ctx sdk.Context) []*types.Pool {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.PoolKey)

	var pools []*types.Pool
	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()

		var pool types.Pool
		k.codec.MustUnmarshal(bz, &pool)

		pools = append(pools, &pool)
	}

	return pools
}
