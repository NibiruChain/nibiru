package keeper

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/MatrixDao/matrix/x/vamm/types"
)

func NewKeeper(codec codec.BinaryCodec, storeKey sdk.StoreKey) Keeper {
	return Keeper{
		codec:    codec,
		storeKey: storeKey,
	}
}

type Keeper struct {
	codec    codec.BinaryCodec
	storeKey sdk.StoreKey
}

// SwapInput swaps pair token
func (k Keeper) SwapInput(
	ctx sdk.Context,
	pair string,
	dir types.Direction,
	quoteAssetAmount sdk.Int,
	baseAmountLimit sdk.Int,
) (sdk.Int, error) {
	if !k.ExistsPool(ctx, pair) {
		return sdk.Int{}, types.ErrPairNotSupported
	}

	if quoteAssetAmount.Equal(sdk.ZeroInt()) {
		return sdk.ZeroInt(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Int{}, err
	}

	if dir == types.Direction_REMOVE_FROM_AMM {
		enoughReserve, err := pool.HasEnoughQuoteReserve(quoteAssetAmount)
		if err != nil {
			return sdk.Int{}, err
		}
		if !enoughReserve {
			return sdk.Int{}, types.ErrOvertradingLimit
		}
	}

	baseAssetAmount, err := pool.GetBaseAmountByQuoteAmount(dir, quoteAssetAmount)
	if err != nil {
		return sdk.Int{}, err
	}

	if !baseAmountLimit.Equal(sdk.ZeroInt()) {
		if dir == types.Direction_ADD_TO_AMM {
			if baseAssetAmount.LT(baseAmountLimit) {
				return sdk.Int{}, fmt.Errorf(
					"base amount (%s) is less than selected limit (%s)",
					baseAssetAmount.String(),
					baseAmountLimit.String(),
				)
			}
		} else {
			if baseAssetAmount.GT(baseAmountLimit) {
				return sdk.Int{}, fmt.Errorf(
					"base amount (%s) is greater than selected limit (%s)",
					baseAssetAmount.String(),
					baseAmountLimit.String(),
				)
			}
		}
	}

	err = k.updateReserve(ctx, pool, dir, quoteAssetAmount, baseAssetAmount)

	return baseAssetAmount, nil
}

// getPool returns the pool from database
func (k Keeper) getPool(ctx sdk.Context, pair string) (*types.Pool, error) {
	store := k.getStore(ctx)

	bz := store.Get(types.GetPoolKey(pair))
	var pool types.Pool

	err := k.codec.Unmarshal(bz, &pool)
	if err != nil {
		return nil, err
	}

	return &pool, nil
}

// CreatePool creates a pool for a specific pair.
func (k Keeper) CreatePool(
	ctx sdk.Context,
	pair string,
	tradeLimitRatio sdk.Dec, // integer with 6 decimals, 1_000_000 means 1.0
	quoteAssetReserve sdk.Int,
	baseAssetReserve sdk.Int,
) error {
	pool := &types.Pool{
		Pair:              pair,
		TradeLimitRatio:   tradeLimitRatio.String(),
		QuoteAssetReserve: quoteAssetReserve.String(),
		BaseAssetReserve:  baseAssetReserve.String(),
	}

	err := k.savePool(ctx, pool)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) savePool(
	ctx sdk.Context,
	pool *types.Pool,
) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.codec.Marshal(pool)
	if err != nil {
		return err
	}

	store.Set(types.GetPoolKey(pool.Pair), bz)

	return nil
}

func (k Keeper) updateReserve(
	ctx sdk.Context,
	pool *types.Pool,
	dir types.Direction,
	quoteAssetAmount sdk.Int,
	baseAssetAmount sdk.Int,
) error {
	if dir == types.Direction_ADD_TO_AMM {
		pool.IncreaseQuoteAssetReserve(quoteAssetAmount)
		pool.DecreaseBaseAssetReserve(baseAssetAmount)
		// TODO baseAssetDeltaThisFunding
		// TODO totalPositionSize
		// TODO cumulativeNotional
	} else {
		pool.DecreaseQuoteAssetReserve(quoteAssetAmount)
		pool.IncreaseBaseAssetReserve(baseAssetAmount)
		// TODO baseAssetDeltaThisFunding
		// TODO totalPositionSize
		// TODO cumulativeNotional
	}

	// TODO check Fluctuation Limit

	return k.savePool(ctx, pool)
}

// ExistsPool returns true if pool exists, false if not.
func (k Keeper) ExistsPool(ctx sdk.Context, pair string) bool {
	store := k.getStore(ctx)
	return store.Has(types.GetPoolKey(pair))
}

func (k Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}
