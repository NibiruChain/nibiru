package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func NewKeeper(
	codec codec.BinaryCodec,
	storeKey sdk.StoreKey,
	pricefeedKeeper types.PricefeedKeeper,
) Keeper {
	return Keeper{
		codec:           codec,
		storeKey:        storeKey,
		pricefeedKeeper: pricefeedKeeper,
	}
}

type Keeper struct {
	codec           codec.BinaryCodec
	storeKey        sdk.StoreKey
	pricefeedKeeper types.PricefeedKeeper
}

//CalcPerpTxFee calculates the total tx fee for exchanging 'quoteAmt' of tokens on
//the exchange.
//
//Args:
//  quoteAmt (sdk.Int):
//
//Returns:
//  toll (sdk.Int): Amount of tokens transferred to the the fee pool.
//  spread (sdk.Int): Amount of tokens transferred to the PerpEF.
//
func (k Keeper) CalcPerpTxFee(ctx sdk.Context, pair common.TokenPair, quoteAmt sdk.Int) (toll sdk.Int, spread sdk.Int, err error) {
	//TODO implement me
	panic("implement me")
}

/*
Trades baseAssets in exchange for quoteAssets.
The base asset is a crypto asset like BTC.
The quote asset is a stablecoin like NUSD.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - baseAssetAmount: the amount of quote asset being traded
  - quoteAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage

ret:
  - quoteAssetAmount: the amount of quote asset swapped
  - err: error
*/
func (k Keeper) SwapBaseForQuote(
	ctx sdk.Context,
	pair common.TokenPair,
	dir types.Direction,
	baseAssetAmount sdk.Dec,
	quoteAmountLimit sdk.Dec,
) (quoteAssetAmount sdk.Dec, err error) {
	if !k.ExistsPool(ctx, pair) {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	if baseAssetAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == types.Direction_REMOVE_FROM_POOL &&
		!pool.HasEnoughBaseReserve(baseAssetAmount) {
		return sdk.Dec{}, types.ErrOverTradingLimit
	}

	quoteAssetAmount, err = pool.GetQuoteAmountByBaseAmount(dir, baseAssetAmount)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !quoteAmountLimit.IsZero() {
		// if going long and the base amount retrieved from the pool is less than the limit
		if dir == types.Direction_ADD_TO_POOL && quoteAssetAmount.LT(quoteAmountLimit) {
			return sdk.Dec{}, fmt.Errorf(
				"quote amount (%s) is less than selected limit (%s)",
				quoteAssetAmount.String(),
				quoteAmountLimit.String(),
			)
			// if going short and the base amount retrieved from the pool is greater than the limit
		} else if dir == types.Direction_REMOVE_FROM_POOL && quoteAssetAmount.GT(quoteAmountLimit) {
			return sdk.Dec{}, fmt.Errorf(
				"quote amount (%s) is greater than selected limit (%s)",
				quoteAssetAmount.String(),
				quoteAmountLimit.String(),
			)
		}
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.IncreaseBaseAssetReserve(baseAssetAmount)
		pool.DecreaseQuoteAssetReserve(quoteAssetAmount)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.DecreaseBaseAssetReserve(baseAssetAmount)
		pool.IncreaseQuoteAssetReserve(quoteAssetAmount)
	}

	if err = k.savePoolAndSnapshot(ctx, pool, false /*skipFluctuationCheck*/); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventSwapBaseForQuote,
			sdk.NewAttribute(types.AttributeQuoteAssetAmount, baseAssetAmount.String()),
			sdk.NewAttribute(types.AttributeBaseAssetAmount, quoteAssetAmount.String()),
		),
	)

	return quoteAssetAmount, nil
}

/*
Trades quoteAssets in exchange for baseAssets.
The quote asset is a stablecoin like NUSD.
The base asset is a crypto asset like BTC or ETH.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - quoteAssetAmount: the amount of quote asset being traded
  - baseAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage

ret:
  - baseAssetAmount: the amount of base asset swapped
  - err: error
*/
func (k Keeper) SwapQuoteForBase(
	ctx sdk.Context,
	pair common.TokenPair,
	dir types.Direction,
	quoteAssetAmount sdk.Dec,
	baseAmountLimit sdk.Dec,
) (baseAssetAmount sdk.Dec, err error) {
	if !k.ExistsPool(ctx, pair) {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	if quoteAssetAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == types.Direction_REMOVE_FROM_POOL &&
		!pool.HasEnoughQuoteReserve(quoteAssetAmount) {
		return sdk.Dec{}, types.ErrOverTradingLimit
	}

	baseAssetAmount, err = pool.GetBaseAmountByQuoteAmount(dir, quoteAssetAmount)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !baseAmountLimit.IsZero() {
		// if going long and the base amount retrieved from the pool is less than the limit
		if dir == types.Direction_ADD_TO_POOL && baseAssetAmount.LT(baseAmountLimit) {
			return sdk.Dec{}, types.ErrAssetOverUserLimit.Wrapf(
				"base amount (%s) is less than selected limit (%s)",
				baseAssetAmount.String(),
				baseAmountLimit.String(),
			)
			// if going short and the base amount retrieved from the pool is greater than the limit
		} else if dir == types.Direction_REMOVE_FROM_POOL && baseAssetAmount.GT(baseAmountLimit) {
			return sdk.Dec{}, types.ErrAssetOverUserLimit.Wrapf(
				"base amount (%s) is greater than selected limit (%s)",
				baseAssetAmount.String(),
				baseAmountLimit.String(),
			)
		}
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.DecreaseBaseAssetReserve(baseAssetAmount)
		pool.IncreaseQuoteAssetReserve(quoteAssetAmount)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.IncreaseBaseAssetReserve(baseAssetAmount)
		pool.DecreaseQuoteAssetReserve(quoteAssetAmount)
	}

	if err = k.savePoolAndSnapshot(ctx, pool, false /*skipFluctuationCheck*/); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventSwapQuoteForBase,
			sdk.NewAttribute(types.AttributeQuoteAssetAmount, quoteAssetAmount.String()),
			sdk.NewAttribute(types.AttributeBaseAssetAmount, baseAssetAmount.String()),
		),
	)

	return baseAssetAmount, nil
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
	price := pool.QuoteAssetReserve.Quo(pool.BaseAssetReserve)

	lastPrice := snapshot.QuoteAssetReserve.Quo(snapshot.BaseAssetReserve)
	upperLimit := lastPrice.Mul(sdk.OneDec().Add(pool.FluctuationLimitRatio))
	lowerLimit := lastPrice.Mul(sdk.OneDec().Sub(pool.FluctuationLimitRatio))

	if price.GT(upperLimit) || price.LT(lowerLimit) {
		return true
	}

	return false
}

func (k Keeper) IsOverSpreadLimit(ctx sdk.Context, pair common.TokenPair) (isIt bool) {
	spotPrice, err := k.GetSpotPrice(ctx, pair)
	if err != nil {
		panic(err)
	}

	oraclePrice, err := k.GetUnderlyingPrice(ctx, pair)
	if err != nil {
		panic(err)
	}

	pool, err := k.getPool(ctx, pair)
	if err != nil {
		panic(err)
	}
	return spotPrice.Sub(oraclePrice).Quo(oraclePrice).Abs().GTE(pool.MaxOracleSpreadRatio)
}
