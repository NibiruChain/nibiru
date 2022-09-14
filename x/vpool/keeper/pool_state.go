package keeper

import (
	"fmt"
	"time"

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
	maintenanceMarginRatio sdk.Dec,
	maxLeverage sdk.Dec,
) {
	pool := types.NewPool(
		pair,
		tradeLimitRatio,
		quoteAssetReserve,
		baseAssetReserve,
		fluctuationLimitRatio,
		maxOracleSpreadRatio,
		maintenanceMarginRatio,
		maxLeverage,
	)

	k.savePool(ctx, pool)
	k.SaveSnapshot(ctx, pair, pool.QuoteAssetReserve, pool.BaseAssetReserve)
}

// getPool returns the pool from database
func (k Keeper) getPool(ctx sdk.Context, pair common.AssetPair) (
	*types.Pool, error,
) {
	bz := ctx.KVStore(k.storeKey).Get(types.GetPoolKey(pair))
	if bz == nil {
		return nil, fmt.Errorf("could not find vpool for pair %s", pair.String())
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
  - skipFluctuationCheck: determines if a fluctuation check should be done against the last snapshot

ret:
  - err: error
*/
func (k Keeper) updatePool(
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
	iterator := sdk.KVStorePrefixIterator(store, types.PoolKeyPrefix)

	var pools []*types.Pool
	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()

		var pool types.Pool
		k.codec.MustUnmarshal(bz, &pool)

		pools = append(pools, &pool)
	}

	return pools
}

// GetPoolPrices returns the mark price, twap (mark) price, and index price for a vpool.
// An error is returned if
func (k Keeper) GetPoolPrices(
	ctx sdk.Context, pool types.Pool,
) (prices types.PoolPrices, err error) {
	// Validation - guarantees no panics in GetUnderlyingPrice or GetCurrentTWAP
	if err := pool.Pair.Validate(); err != nil {
		return prices, err
	}
	if !k.ExistsPool(ctx, pool.Pair) {
		return prices, types.ErrPairNotSupported.Wrap(pool.Pair.String())
	}
	if err := pool.ValidateReserves(); err != nil {
		return prices, err
	}

	indexPrice, err := k.GetUnderlyingPrice(ctx, pool.Pair)
	if err != nil {
		// fail gracefully so that vpool queries run even if the oracle price feeds stop
		k.Logger(ctx).Error(err.Error())
	}

	twapMark, err := k.calcTwap(
		ctx,
		pool.Pair,
		types.TwapCalcOption_SPOT,
		types.Direction_DIRECTION_UNSPECIFIED,
		sdk.ZeroDec(),
		15*time.Minute,
	)
	if err != nil {
		// fail gracefully so that vpool queries run even if the TWAP is undefined.
		k.Logger(ctx).Error(err.Error())
	}

	return types.PoolPrices{
		Pair:          pool.Pair.String(),
		MarkPrice:     pool.QuoteAssetReserve.Quo(pool.BaseAssetReserve),
		TwapMark:      twapMark.String(),
		IndexPrice:    indexPrice.String(),
		SwapInvariant: pool.BaseAssetReserve.Mul(pool.QuoteAssetReserve).RoundInt(),
		BlockNumber:   ctx.BlockHeight(),
	}, nil
}
