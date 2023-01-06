package keeper

import (
	"fmt"
	"math"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// CreatePool creates a pool for a specific pair.
func (k Keeper) CreatePool(
	ctx sdk.Context,
	pair common.AssetPair,
	quoteAssetReserve sdk.Dec,
	baseAssetReserve sdk.Dec,
	config types.VpoolConfig,
) {
	vpool := types.Vpool{
		Pair:              pair,
		BaseAssetReserve:  baseAssetReserve,
		QuoteAssetReserve: quoteAssetReserve,
		Config:            config,
	}
	k.Pools.Insert(ctx, pair, vpool)

	k.ReserveSnapshots.Insert(
		ctx,
		collections.Join(pair, ctx.BlockTime()),
		vpool.ToSnapshot(ctx),
	)
}

func (k Keeper) EditPoolConfig(
	ctx sdk.Context,
	pair common.AssetPair,
	config types.VpoolConfig,
) error {
	// Grab current pool from state
	vpool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return err
	}

	newVpool := types.Vpool{
		Pair:              vpool.Pair,
		BaseAssetReserve:  vpool.BaseAssetReserve,
		QuoteAssetReserve: vpool.QuoteAssetReserve,
		Config:            config, // main change is here
	}
	if err := newVpool.Validate(); err != nil {
		return err
	}

	err = k.updatePool(
		ctx,
		newVpool,
		/*skipFluctuationLimitCheck*/ true)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) EditSwapInvariant(
	ctx sdk.Context,
	swapInvariantMap types.EditSwapInvariantsProposal_SwapInvariantMultiple,
) error {
	if err := swapInvariantMap.Validate(); err != nil {
		return err
	}

	// Grab current pool from state
	vpool, err := k.Pools.Get(ctx, common.MustNewAssetPair(swapInvariantMap.Pair))
	if err != nil {
		return err
	}

	// price = y / x
	// k = x * y
	// newK = (cx) * (cy) = c^2 xy = c^2 k
	// newPrice = (c y) / (c x) = y / x = price
	swapInvariant := vpool.BaseAssetReserve.Mul(vpool.QuoteAssetReserve)
	newSwapInvariant := swapInvariant.Mul(swapInvariantMap.Multiplier)

	// Change the swap invariant while holding price constant.
	// Multiplying by the same factor to both of the reserves won't affect price.
	cSquared := newSwapInvariant.Quo(swapInvariant).MustFloat64()
	cAsFloat := math.Sqrt(cSquared)
	c, err := sdk.NewDecFromStr(fmt.Sprintf("%f", cAsFloat))
	if err != nil {
		return err
	}
	newBaseAmount := c.Mul(vpool.BaseAssetReserve)
	newQuoteAmount := c.Mul(vpool.QuoteAssetReserve)

	newVpool := types.Vpool{
		Pair:              vpool.Pair,
		BaseAssetReserve:  newBaseAmount,
		QuoteAssetReserve: newQuoteAmount,
		Config:            vpool.Config,
	}
	if err := newVpool.Validate(); err != nil {
		return err
	}

	err = k.updatePool(
		ctx,
		newVpool,
		/*skipFluctuationLimitCheck*/ true)
	if err != nil {
		return err
	}
	return nil
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
	updatedPool types.Vpool,
	skipFluctuationCheck bool,
) (err error) {
	if !skipFluctuationCheck {
		if err = k.checkFluctuationLimitRatio(ctx, updatedPool); err != nil {
			return err
		}
	}

	k.Pools.Insert(ctx, updatedPool.Pair, updatedPool)

	return nil
}

// ExistsPool returns true if pool exists, false if not.
func (k Keeper) ExistsPool(ctx sdk.Context, pair common.AssetPair) bool {
	_, err := k.Pools.Get(ctx, pair)
	return err == nil
}

// GetPoolPrices returns the mark price, twap (mark) price, and index price for a vpool.
// An error is returned if the pool does not exist.
// No error is returned if the prices don't exist, however.
func (k Keeper) GetPoolPrices(
	ctx sdk.Context, pool types.Vpool,
) (types.PoolPrices, error) {
	if err := pool.Pair.Validate(); err != nil {
		return types.PoolPrices{}, err
	}

	if !k.ExistsPool(ctx, pool.Pair) {
		return types.PoolPrices{}, types.ErrPairNotSupported.Wrap(pool.Pair.String())
	}

	if err := pool.ValidateReserves(); err != nil {
		return types.PoolPrices{}, err
	}

	indexPrice, err := k.oracleKeeper.GetExchangeRate(ctx, pool.Pair.String())
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
