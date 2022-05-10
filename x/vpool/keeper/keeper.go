package keeper

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k Keeper) GetOpenInterestNotionalCap(ctx sdk.Context, pair common.TokenPair) (sdk.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (k Keeper) GetMaxHoldingBaseAsset(ctx sdk.Context, pair common.TokenPair) (sdk.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (k Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

// SwapInput swaps pair token
func (k Keeper) SwapInput(
	ctx sdk.Context,
	pair common.TokenPair,
	dir types.Direction,
	quoteAssetAmount sdk.Int,
	baseAmountLimit sdk.Int,
) (sdk.Int, error) {
	if !k.existsPool(ctx, pair) {
		return sdk.Int{}, types.ErrPairNotSupported
	}

	if quoteAssetAmount.Equal(sdk.ZeroInt()) {
		return sdk.ZeroInt(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Int{}, err
	}

	if dir == types.Direction_REMOVE_FROM_POOL {
		enoughReserve := pool.HasEnoughQuoteReserve(quoteAssetAmount)
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
		if dir == types.Direction_ADD_TO_POOL {
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

	err = k.updateReserve(ctx, pool, dir, quoteAssetAmount, baseAssetAmount, false)
	if err != nil {
		return sdk.Int{}, fmt.Errorf("error updating reserve: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventSwapInput,
			sdk.NewAttribute(types.AttributeToken0Amount, quoteAssetAmount.String()),
			sdk.NewAttribute(types.AttributeToken1Amount, baseAssetAmount.String()),
		),
	)

	return baseAssetAmount, nil
}

// getPool returns the pool from database
func (k Keeper) getPool(ctx sdk.Context, pair common.TokenPair) (*types.Pool, error) {
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
	fluctuationLimitRation sdk.Dec,
) error {
	pool := types.NewPool(pair, tradeLimitRatio, quoteAssetReserve, baseAssetReserve, fluctuationLimitRation)

	err := k.savePool(ctx, pool)
	if err != nil {
		return err
	}

	err = k.saveReserveSnapshot(ctx, 0, pool)
	if err != nil {
		return fmt.Errorf("error saving snapshot on pool creation: %w", err)
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

	store.Set(types.GetPoolKey(common.TokenPair(pool.Pair)), bz)

	return nil
}

func (k Keeper) updateReserve(
	ctx sdk.Context,
	pool *types.Pool,
	dir types.Direction,
	quoteAssetAmount sdk.Int,
	baseAssetAmount sdk.Int,
	skipFluctuationCheck bool,
) error {
	if dir == types.Direction_ADD_TO_POOL {
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

	// Check if its over Fluctuation Limit Ratio.
	if !skipFluctuationCheck {
		err := k.checkFluctuationLimitRatio(ctx, pool)
		if err != nil {
			return err
		}
	}

	err := k.addReserveSnapshot(ctx, pool)
	if err != nil {
		return fmt.Errorf("error creating snapshot: %w", err)
	}

	return k.savePool(ctx, pool)
}

// existsPool returns true if pool exists, false if not.
func (k Keeper) existsPool(ctx sdk.Context, pair common.TokenPair) bool {
	store := k.getStore(ctx)
	return store.Has(types.GetPoolKey(pair))
}

func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool *types.Pool) error {
	if pool.FluctuationLimitRatio.GT(sdk.ZeroDec()) {
		latestSnapshot, counter, err := k.getLastReserveSnapshot(ctx, common.TokenPair(pool.Pair))
		if err != nil {
			return fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
		}

		if latestSnapshot.BlockNumber == ctx.BlockHeight() && counter > 1 {
			latestSnapshot, err = k.getSnapshotByCounter(ctx, counter-1)
			if err != nil {
				return fmt.Errorf("error getting snapshot number %d from pair %s", counter-1, pool.Pair)
			}
		}

		if isOverFluctuationLimit(pool, latestSnapshot) {
			return types.ErrOverFluctuationLimit
		}
	}

	return nil
}

func isOverFluctuationLimit(pool *types.Pool, snapshot types.ReserveSnapshot) bool {
	price := pool.QuoteAssetReserve.ToDec().Quo(pool.BaseAssetReserve.ToDec())

	snapshotQuote, _ := sdk.NewDecFromStr(snapshot.Token1Reserve)
	snapshotBase, _ := sdk.NewDecFromStr(snapshot.Token0Reserve)
	lastPrice := snapshotQuote.Quo(snapshotBase)
	upperLimit := lastPrice.Mul(sdk.OneDec().Add(pool.FluctuationLimitRatio))
	lowerLimit := lastPrice.Mul(sdk.OneDec().Sub(pool.FluctuationLimitRatio))

	if price.GT(upperLimit) || price.LT(lowerLimit) {
		return true
	}

	return false
}
