package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/vamm/types"
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

func (k Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

// SwapInput swaps pair token
func (k Keeper) SwapInput(
	ctx sdk.Context,
	pair string,
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

	err = k.updateReserve(ctx, pool, dir, quoteAssetAmount, baseAssetAmount, false)
	if err != nil {
		return sdk.Int{}, fmt.Errorf("error updating reserve: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventSwapInput,
			sdk.NewAttribute(types.AttributeQuoteAssetAmount, quoteAssetAmount.String()),
			sdk.NewAttribute(types.AttributeBaseAssetAmount, baseAssetAmount.String()),
		),
	)

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

	store.Set(types.GetPoolKey(pool.Pair), bz)

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
func (k Keeper) existsPool(ctx sdk.Context, pair string) bool {
	store := k.getStore(ctx)
	return store.Has(types.GetPoolKey(pair))
}

func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool *types.Pool) error {
	fluctuationLimitRatio, err := sdk.NewDecFromStr(pool.FluctuationLimitRatio)
	if err != nil {
		return fmt.Errorf("error getting fluctuation limit ratio for pool: %s", pool.Pair)
	}

	if fluctuationLimitRatio.GT(sdk.ZeroDec()) {
		latestSnapshot, counter, err := k.getLastReserveSnapshot(ctx, pool.Pair)
		if err != nil {
			return fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
		}

		if latestSnapshot.BlockNumber == ctx.BlockHeight() && counter > 1 {
			latestSnapshot, err = k.getSnapshotByCounter(ctx, pool.Pair, counter-1)
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
	fluctuationLimitRatio, _ := sdk.NewDecFromStr(pool.FluctuationLimitRatio)
	quoteAssetReserve, _ := pool.GetPoolQuoteAssetReserveAsInt()
	baseAssetReserve, _ := pool.GetPoolBaseAssetReserveAsInt()
	price := quoteAssetReserve.ToDec().Quo(baseAssetReserve.ToDec())

	snapshotQuote, _ := sdk.NewDecFromStr(snapshot.QuoteAssetReserve)
	snapshotBase, _ := sdk.NewDecFromStr(snapshot.BaseAssetReserve)
	lastPrice := snapshotQuote.Quo(snapshotBase)
	upperLimit := lastPrice.Mul(sdk.OneDec().Add(fluctuationLimitRatio))
	lowerLimit := lastPrice.Mul(sdk.OneDec().Sub(fluctuationLimitRatio))

	if price.GT(upperLimit) || price.LT(lowerLimit) {
		return true
	}

	return false
}
