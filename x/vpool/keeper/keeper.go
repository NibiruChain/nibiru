package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/nibiru/collections"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

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
		Pools:           collections.NewMap(storeKey, 0, common.AssetPairKeyEncoder, collections.ProtoValueEncoder[types.VPool](codec)),
		ReserveSnapshots: collections.NewMap(
			storeKey, 1,
			collections.PairKeyEncoder(common.AssetPairKeyEncoder, collections.TimeKeyEncoder),
			collections.ProtoValueEncoder[types.ReserveSnapshot](codec),
		),
	}
}

type Keeper struct {
	codec           codec.BinaryCodec
	storeKey        sdk.StoreKey
	pricefeedKeeper types.PricefeedKeeper

	Pools            collections.Map[common.AssetPair, types.VPool]
	ReserveSnapshots collections.Map[collections.Pair[common.AssetPair, time.Time], types.ReserveSnapshot]
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

/*
SwapBaseForQuote
Trades baseAssets in exchange for quoteAssets.
The base asset is a crypto asset like BTC.
The quote asset is a stablecoin like NUSD.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - baseAssetAmount: the amount of quote asset being traded
  - quoteAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - quoteAssetAmount: the amount of quote asset swapped
  - err: error
*/
func (k Keeper) SwapBaseForQuote(
	ctx sdk.Context,
	pair common.AssetPair,
	dir types.Direction,
	baseAmt sdk.Dec,
	quoteLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (quoteAmt sdk.Dec, err error) {
	if baseAmt.IsZero() {
		return sdk.ZeroDec(), nil
	}

	if !k.pricefeedKeeper.IsActivePair(ctx, pair.String()) {
		return sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", pair.String())
	}

	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	if !pool.HasEnoughBaseReserve(baseAmt) {
		return sdk.Dec{}, types.ErrOverTradingLimit
	}

	quoteAmt, err = pool.GetQuoteAmountByBaseAmount(dir, baseAmt)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !pool.HasEnoughQuoteReserve(quoteAmt) {
		// in reality, this if statement should never run because a perturbation in quote reserve assets
		// greater than trading limit ratio would have happened when checking for a perturbation in
		// base assets, due to x*y=k
		//
		// e.g. a 10% change in quote asset reserves would have triggered a >10% change in
		// base asset reserves
		return sdk.Dec{}, types.ErrOverTradingLimit.Wrapf(
			"quote amount %s is over trading limit", quoteAmt)
	}

	err = checkIfLimitIsViolated(quoteLimit, quoteAmt, dir)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.IncreaseBaseAssetReserve(baseAmt)
		pool.DecreaseQuoteAssetReserve(quoteAmt)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.DecreaseBaseAssetReserve(baseAmt)
		pool.IncreaseQuoteAssetReserve(quoteAmt)
	}

	if err = k.updatePool(ctx, pool, skipFluctuationLimitCheck); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.MarkPriceChangedEvent{
		Pair:      pair.String(),
		Price:     pool.GetMarkPrice(),
		Timestamp: ctx.BlockHeader().Time,
	}); err != nil {
		return sdk.Dec{}, err
	}

	return quoteAmt, ctx.EventManager().EmitTypedEvent(&types.SwapBaseForQuoteEvent{
		Pair:        pair.String(),
		QuoteAmount: quoteAmt,
		BaseAmount:  baseAmt,
	})
}

/*
SwapQuoteForBase
Trades quoteAssets in exchange for baseAssets.
The quote asset is a stablecoin like NUSD.
The base asset is a crypto asset like BTC or ETH.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - quoteAssetAmount: the amount of quote asset being traded
  - baseAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - baseAssetAmount: the amount of base asset swapped
  - err: error
*/
func (k Keeper) SwapQuoteForBase(
	ctx sdk.Context,
	pair common.AssetPair,
	dir types.Direction,
	quoteAmt sdk.Dec,
	baseLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (baseAmt sdk.Dec, err error) {
	if quoteAmt.IsZero() {
		return sdk.ZeroDec(), nil
	}

	if !k.pricefeedKeeper.IsActivePair(ctx, pair.String()) {
		return sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", pair.String())
	}

	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return sdk.Dec{}, types.ErrPairNotSupported
	}

	// check trade limit ratio on quote in either direction
	if !pool.HasEnoughQuoteReserve(quoteAmt) {
		return sdk.Dec{}, types.ErrOverTradingLimit.Wrapf(
			"quote amount %s is over trading limit", quoteAmt)
	}

	baseAmt, err = pool.GetBaseAmountByQuoteAmount(dir, quoteAmt)
	if err != nil {
		return sdk.Dec{}, err
	}

	if !pool.HasEnoughBaseReserve(baseAmt) {
		// in reality, this if statement should never run because a perturbation in base reserve assets
		// greater than trading limit ratio would have happened when checking for a perturbation in
		// quote assets, due to x*y=k
		//
		// e.g. a 10% change in base asset reserves would have triggered a >10% change in
		// quote asset reserves
		return sdk.Dec{}, types.ErrOverTradingLimit.Wrapf(
			"base amount %s is over trading limit", baseAmt)
	}

	err = checkIfLimitIsViolated(baseLimit, baseAmt, dir)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == types.Direction_ADD_TO_POOL {
		pool.DecreaseBaseAssetReserve(baseAmt)
		pool.IncreaseQuoteAssetReserve(quoteAmt)
	} else if dir == types.Direction_REMOVE_FROM_POOL {
		pool.IncreaseBaseAssetReserve(baseAmt)
		pool.DecreaseQuoteAssetReserve(quoteAmt)
	}

	if err = k.updatePool(ctx, pool, skipFluctuationLimitCheck); err != nil {
		return sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.MarkPriceChangedEvent{
		Pair:      pair.String(),
		Price:     pool.GetMarkPrice(),
		Timestamp: ctx.BlockHeader().Time,
	}); err != nil {
		return sdk.Dec{}, err
	}

	return baseAmt, ctx.EventManager().EmitTypedEvent(&types.SwapQuoteForBaseEvent{
		Pair:        pair.String(),
		QuoteAmount: quoteAmt,
		BaseAmount:  baseAmt,
	})
}

// checkIfLimitIsViolated checks if the limit is violated by the amount.
// returns error if it does
func checkIfLimitIsViolated(limit, amount sdk.Dec, dir types.Direction) error {
	if !limit.IsZero() {
		if dir == types.Direction_ADD_TO_POOL && amount.LT(limit) {
			return types.ErrAssetFailsUserLimit.Wrapf(
				"amount (%s) is less than selected limit (%s)",
				amount.String(),
				limit.String(),
			)
		} else if dir == types.Direction_REMOVE_FROM_POOL && amount.GT(limit) {
			return types.ErrAssetFailsUserLimit.Wrapf(
				"amount (%s) is greater than selected limit (%s)",
				amount.String(),
				limit.String(),
			)
		}
	}

	return nil
}

/*
*
Check's that a pool that we're about to save to state does not violate the fluctuation limit.
Always tries to check against a snapshot from a previous block. If one doesn't exist, then it just uses the current snapshot.
This should run prior to updating the snapshot, otherwise it will compare the currently updated vpool to itself.

args:
  - ctx: the cosmos-sdk context
  - pool: the updated vpool

ret:
  - err: error if any
*/
func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool types.VPool) error {
	if pool.FluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[common.AssetPair, time.Time]{}.Prefix(pool.Pair).Descending())
	defer it.Close()
	if !it.Valid() {
		return fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
	}
	latestSnapshot := it.Value()
	if pool.IsOverFluctuationLimitInRelationWithSnapshot(latestSnapshot) {
		return types.ErrOverFluctuationLimit
	}

	return nil
}

/*
IsOverSpreadLimit compares the current spot price of the vpool (given by pair) to the underlying's index price (given by an oracle).
It panics if you provide it with a pair that doesn't exist in the state.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - bool: whether the price has deviated from the oracle price beyond a spread ratio
*/
func (k Keeper) IsOverSpreadLimit(ctx sdk.Context, pair common.AssetPair) bool {
	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		panic(err)
	}

	indexPrice, err := k.pricefeedKeeper.GetCurrentPrice(ctx, pair.BaseDenom(), pair.QuoteDenom())
	if err != nil {
		panic(err)
	}

	return pool.IsOverSpreadLimit(indexPrice.Price)
}

/*
GetMaintenanceMarginRatio returns the maintenance margin ratio for the pool from the asset pair.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - sdk.Dec: The maintenance margin ratio for the pool
*/
func (k Keeper) GetMaintenanceMarginRatio(ctx sdk.Context, pair common.AssetPair) sdk.Dec {
	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		panic(err)
	}

	return pool.MaintenanceMarginRatio
}

/*
GetMaxLeverage returns the maximum leverage required to open a position in the pool.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - sdk.Dec: The maintenance margin ratio for the pool
*/
func (k Keeper) GetMaxLeverage(ctx sdk.Context, pair common.AssetPair) sdk.Dec {
	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		panic(err)
	}

	return pool.MaxLeverage
}

/*
GetAllPools returns an array of all the pools

args:
  - ctx: the cosmos-sdk context

ret:
  - []types.VPool: All defined vpool
*/
func (k Keeper) GetAllPools(ctx sdk.Context) []types.VPool {
	return k.Pools.Iterate(ctx, collections.Range[common.AssetPair]{}).Values()
}
