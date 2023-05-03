package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

// CreatePool creates a pool for a specific pair.
func (k Keeper) CreatePool(
	ctx sdk.Context,
	pair asset.Pair,
	quoteReserve sdk.Dec,
	baseReserve sdk.Dec,
	config types.MarketConfig,
	pegMultiplier sdk.Dec,
) error {
	if !quoteReserve.Equal(baseReserve) {
		return fmt.Errorf("quote asset reserve %s must be equal to base asset reserve %s", quoteReserve, baseReserve)
	}

	market := types.NewMarket(types.ArgsNewMarket{
		Pair:          pair,
		BaseReserves:  baseReserve,
		QuoteReserves: quoteReserve,
		Config:        &config,
		Bias:          sdk.ZeroDec(),
		PegMultiplier: pegMultiplier,
	})

	if !baseReserve.Equal(market.SqrtDepth) {
		return fmt.Errorf(
			"base asset reserve %s must be equal to sqrt depth %s on pool creation",
			baseReserve, market.SqrtDepth,
		)
	}

	err := market.Validate()
	if err != nil {
		return err
	}

	return common.TryCatch(func() {
		k.Pools.Insert(ctx, pair, market)

		k.ReserveSnapshots.Insert(
			ctx,
			collections.Join(pair, ctx.BlockTime()),
			market.ToSnapshot(ctx),
		)
	})()
}

func (k Keeper) EditPoolPegMultiplier(
	ctx sdk.Context,
	pair asset.Pair,
	newPeg sdk.Dec,
) error {
	// Grab current pool from state
	market, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return err
	}

	market.PegMultiplier = newPeg
	if err := market.Validate(); err != nil {
		return err
	}

	err = k.updatePool(
		ctx,
		market,
		/*skipFluctuationLimitCheck*/ true)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) EditPoolConfig(
	ctx sdk.Context,
	pair asset.Pair,
	config types.MarketConfig,
) error {
	// Grab current pool from state
	market, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return err
	}

	newMarket := types.Market{
		Pair:          market.Pair,
		BaseReserve:   market.BaseReserve,
		QuoteReserve:  market.QuoteReserve,
		SqrtDepth:     market.SqrtDepth,
		PegMultiplier: market.PegMultiplier,
		Config:        config, // main change is here
	}
	if err := newMarket.Validate(); err != nil {
		return err
	}

	err = k.updatePool(
		ctx,
		newMarket,
		/*skipFluctuationLimitCheck*/ true)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) EditSwapInvariant(
	ctx sdk.Context,
	pair asset.Pair,
	swapInvariantMultiplier sdk.Dec,
) error {
	// Grab current pool from state
	market, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return err
	}

	newMarket, err := market.UpdateSwapInvariant(swapInvariantMultiplier)
	if err != nil {
		return err
	}

	err = k.updatePool(
		ctx,
		newMarket,
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
	updatedPool types.Market,
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
func (k Keeper) ExistsPool(ctx sdk.Context, pair asset.Pair) bool {
	_, err := k.Pools.Get(ctx, pair)
	return err == nil
}

// GetPoolPrices returns the mark price, twap (mark) price, and index price for a market.
// An error is returned if the pool does not exist.
// No error is returned if the prices don't exist, however.
func (k Keeper) GetPoolPrices(
	ctx sdk.Context, pool types.Market,
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

	indexPrice, err := k.oracleKeeper.GetExchangeRate(ctx, pool.Pair)
	if err != nil {
		// fail gracefully so that market queries run even if the oracle price feeds stop
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
		// fail gracefully so that market queries run even if the TWAP is undefined.
		k.Logger(ctx).Error(err.Error())
	}

	return types.PoolPrices{
		Pair:          pool.Pair,
		MarkPrice:     pool.QuoteReserve.Quo(pool.BaseReserve).Mul(pool.PegMultiplier),
		TwapMark:      twapMark.String(),
		IndexPrice:    indexPrice.String(),
		SwapInvariant: pool.BaseReserve.Mul(pool.QuoteReserve).RoundInt(),
		BlockNumber:   ctx.BlockHeight(),
	}, nil
}
