package keeper

import (
	"time"

	"github.com/NibiruChain/nibiru/collections/keys"

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
	k.Pools.Insert(ctx, pair, types.VPool{
		Pair:                   pair,
		BaseAssetReserve:       baseAssetReserve,
		QuoteAssetReserve:      quoteAssetReserve,
		TradeLimitRatio:        tradeLimitRatio,
		FluctuationLimitRatio:  fluctuationLimitRatio,
		MaxOracleSpreadRatio:   maxOracleSpreadRatio,
		MaintenanceMarginRatio: maintenanceMarginRatio,
		MaxLeverage:            maxLeverage,
	})

	k.ReserveSnapshots.Insert(
		ctx,
		keys.Join(pair, keys.Uint64(uint64(ctx.BlockTime().UnixMilli()))),
		types.NewReserveSnapshot(
			pair,
			baseAssetReserve,
			quoteAssetReserve,
			ctx.BlockTime(),
		),
	)
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
	updatedPool types.VPool,
	skipFluctuationCheck bool,
) (err error) {
	// Check if its over Fluctuation Limit Ratio.
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
	ctx sdk.Context, pool types.VPool,
) (prices types.PoolPrices, err error) {
	// Validation
	if err := pool.Pair.Validate(); err != nil {
		return prices, err
	}
	if !k.ExistsPool(ctx, pool.Pair) {
		return prices, types.ErrPairNotSupported.Wrap(pool.Pair.String())
	}
	if err := pool.ValidateReserves(); err != nil {
		return prices, err
	}

	indexPrice, err := k.pricefeedKeeper.GetCurrentPrice(ctx, pool.Pair.Token0, pool.Pair.Token1)
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
		IndexPrice:    indexPrice.Price.String(),
		SwapInvariant: pool.BaseAssetReserve.Mul(pool.QuoteAssetReserve).RoundInt(),
		BlockNumber:   ctx.BlockHeight(),
	}, nil
}
