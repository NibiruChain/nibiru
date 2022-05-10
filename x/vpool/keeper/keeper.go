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
		if !pool.HasEnoughQuoteReserve(quoteAssetAmount) {
			return sdk.Int{}, types.ErrOvertradingLimit
		}
	}

	baseAssetAmount, err := pool.GetBaseAmountByQuoteAmount(dir, quoteAssetAmount)
	if err != nil {
		return sdk.Int{}, err
	}

	if !baseAmountLimit.IsZero() {
		// if going long and the base amount retrieved from the pool is less than the limit
		if dir == types.Direction_ADD_TO_POOL && baseAssetAmount.LT(baseAmountLimit) {
			return sdk.Int{}, fmt.Errorf(
				"base amount (%s) is less than selected limit (%s)",
				baseAssetAmount.String(),
				baseAmountLimit.String(),
			)
			// if going short and the base amount retrieved from the pool is greater than the limit
		} else if dir == types.Direction_REMOVE_FROM_POOL && baseAssetAmount.GT(baseAmountLimit) {
			return sdk.Int{}, fmt.Errorf(
				"base amount (%s) is greater than selected limit (%s)",
				baseAssetAmount.String(),
				baseAmountLimit.String(),
			)
		}
	}

	if err = k.updateReserve(
		ctx,
		pool,
		dir,
		quoteAssetAmount,
		baseAssetAmount,
		/*skipFluctuationCheck=*/ false,
	); err != nil {
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

// CreatePool creates a pool for a specific pair.
func (k Keeper) CreatePool(
	ctx sdk.Context,
	pair string,
	tradeLimitRatio sdk.Dec, // integer with 6 decimals, 1_000_000 means 1.0
	quoteAssetReserve sdk.Int,
	baseAssetReserve sdk.Int,
	fluctuationLimitRatio sdk.Dec,
) {
	pool := types.NewPool(
		pair,
		tradeLimitRatio,
		quoteAssetReserve,
		baseAssetReserve,
		fluctuationLimitRatio,
	)

	k.savePool(ctx, pool)
	k.saveSnapshot(ctx, pool, 0)
	k.saveSnapshotCounter(ctx, common.TokenPair(pair), 0)
}

func (k Keeper) savePool(
	ctx sdk.Context,
	pool *types.Pool,
) {
	bz := k.codec.MustMarshal(pool)
	ctx.KVStore(k.storeKey).Set(types.GetPoolKey(common.TokenPair(pool.Pair)), bz)
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

	k.savePool(ctx, pool)

	return nil
}

// existsPool returns true if pool exists, false if not.
func (k Keeper) existsPool(ctx sdk.Context, pair common.TokenPair) bool {
	return ctx.KVStore(k.storeKey).Has(types.GetPoolKey(pair))
}

func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool *types.Pool) error {
	if pool.FluctuationLimitRatio.GT(sdk.ZeroDec()) {
		latestSnapshot, counter, err := k.getLatestReserveSnapshot(ctx, common.TokenPair(pool.Pair))
		if err != nil {
			return fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
		}

		if latestSnapshot.BlockNumber == ctx.BlockHeight() && counter > 0 {
			latestSnapshot, err = k.getSnapshot(ctx, common.TokenPair(pool.Pair), counter-1)
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

	lastPrice := snapshot.QuoteAssetReserve.ToDec().Quo(snapshot.BaseAssetReserve.ToDec())
	upperLimit := lastPrice.Mul(sdk.OneDec().Add(pool.FluctuationLimitRatio))
	lowerLimit := lastPrice.Mul(sdk.OneDec().Sub(pool.FluctuationLimitRatio))

	if price.GT(upperLimit) || price.LT(lowerLimit) {
		return true
	}

	return false
}
