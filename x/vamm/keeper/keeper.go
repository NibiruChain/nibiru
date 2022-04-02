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

func (k Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

// SwapOutput swaps your base asset to quote asset.
func (k Keeper) SwapOutput(
	ctx sdk.Context,
	pair string,
	dir types.Direction,
	baseAssetAmount sdk.Int,
	quoteAssetAmountLimit sdk.Int,
	skipFluctuationCheck bool,
) (sdk.Int, error) {
	if !k.existsPool(ctx, pair) {
		return sdk.Int{}, types.ErrPairNotSupported
	}

	if baseAssetAmount.IsZero() {
		return sdk.ZeroInt(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Int{}, err
	}

	if dir == types.Direction_REMOVE_FROM_AMM {
		enoughReserve, err := pool.HasEnoughBaseReserve(baseAssetAmount)
		if err != nil {
			return sdk.Int{}, err
		}

		if !enoughReserve {
			return sdk.Int{}, types.ErrOvertradingLimit
		}
	}

	quoteAssetAmount, err := pool.GetQuoteAmountByBaseAmount(dir, baseAssetAmount)
	if err != nil {
		return sdk.Int{}, err
	}

	if !quoteAssetAmountLimit.IsZero() {
		if dir == types.Direction_ADD_TO_AMM {
			// SHORT
			if quoteAssetAmount.LT(quoteAssetAmountLimit) {
				return sdk.Int{}, fmt.Errorf(
					"quote amount (%s) is less than selected limit (%s)",
					quoteAssetAmount.String(),
					quoteAssetAmountLimit.String(),
				)
			}
		} else {
			// LONG
			if quoteAssetAmount.GT(quoteAssetAmountLimit) {
				return sdk.Int{}, fmt.Errorf(
					"quote amount (%s) is more than selected limit (%s)",
					quoteAssetAmount.String(),
					quoteAssetAmountLimit.String(),
				)
			}
		}
	}

	// If the price impact of one single tx is larger than priceFluctuation, skip the check
	// only for liquidate()
	if !skipFluctuationCheck {
		skipFluctuationCheck, err = k.isSingleTxOverFluctuation(ctx, dir, pool, quoteAssetAmount, baseAssetAmount)
		if err != nil {
			return sdk.Int{}, err
		}
	}

	// Invert direction to update reserve
	var updateDir types.Direction
	if dir == types.Direction_ADD_TO_AMM {
		updateDir = types.Direction_REMOVE_FROM_AMM
	} else {
		updateDir = types.Direction_ADD_TO_AMM
	}
	err = k.updateReserve(
		ctx,
		pool,
		updateDir,
		quoteAssetAmount,
		baseAssetAmount,
		skipFluctuationCheck,
	)
	if err != nil {
		return sdk.Int{}, fmt.Errorf("error updating reserve: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventSwapOutput,
			sdk.NewAttribute(types.AttributeQuoteAssetAmount, quoteAssetAmount.String()),
			sdk.NewAttribute(types.AttributeBaseAssetAmount, baseAssetAmount.String()),
		),
	)

	return quoteAssetAmount, nil
}

func (k Keeper) isSingleTxOverFluctuation(ctx sdk.Context, dir types.Direction, pool *types.Pool, quoteAssetAmount sdk.Int, baseAssetAmount sdk.Int) (bool, error) {
	quoteAssetReserve, _ := pool.GetPoolQuoteAssetReserveAsInt()
	baseAssetReserve, _ := pool.GetPoolBaseAssetReserveAsInt()

	if dir == types.Direction_ADD_TO_AMM {
		pool.QuoteAssetReserve = quoteAssetReserve.Sub(baseAssetAmount).String()
		pool.BaseAssetReserve = baseAssetReserve.Add(baseAssetAmount).String()
	} else {
		pool.QuoteAssetReserve = quoteAssetReserve.Add(baseAssetAmount).String()
		pool.BaseAssetReserve = baseAssetReserve.Sub(baseAssetAmount).String()
	}

	counter, found := k.getSnapshotCounter(ctx, pool.Pair)
	if !found {
		return false, fmt.Errorf("snapshot not found for pair: %s", pool.Pair)
	}

	snapshot, err := k.getSnapshotByCounter(ctx, pool.Pair, counter-1)
	if err != nil {
		return false, err
	}

	return isOverFluctuationLimit(pool, snapshot), nil
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
