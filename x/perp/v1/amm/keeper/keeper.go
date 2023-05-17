package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
)

func NewKeeper(
	codec codec.BinaryCodec,
	storeKey sdk.StoreKey,
	oracleKeeper types.OracleKeeper,
) Keeper {
	return Keeper{
		codec:        codec,
		storeKey:     storeKey,
		oracleKeeper: oracleKeeper,
		Pools:        collections.NewMap(storeKey, 0, asset.PairKeyEncoder, collections.ProtoValueEncoder[types.Market](codec)),
		ReserveSnapshots: collections.NewMap(
			storeKey, 1,
			collections.PairKeyEncoder(asset.PairKeyEncoder, collections.TimeKeyEncoder),
			collections.ProtoValueEncoder[types.ReserveSnapshot](codec),
		),
	}
}

type Keeper struct {
	codec        codec.BinaryCodec
	storeKey     sdk.StoreKey
	oracleKeeper types.OracleKeeper

	Pools            collections.Map[asset.Pair, types.Market]
	ReserveSnapshots collections.Map[collections.Pair[asset.Pair, time.Time], types.ReserveSnapshot]
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
  - baseAmt: the amount of base asset being traded
  - quoteAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - quoteAmtAbs: the absolute value of the amount swapped in quote assets
  - err: error
*/
func (k Keeper) SwapBaseForQuote(
	ctx sdk.Context,
	market types.Market,
	dir types.Direction,
	baseAmt sdk.Dec,
	quoteLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedMarket types.Market, quoteAssetAmtAbs sdk.Dec, err error) {
	if baseAmt.IsZero() {
		return market, sdk.ZeroDec(), nil
	}

	if _, err = k.oracleKeeper.GetExchangeRate(ctx, market.Pair); err != nil {
		return market, sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", market.Pair)
	}

	baseAmtAbs := baseAmt.Abs()
	quoteReserveAbs, err := market.GetQuoteReserveByBase(baseAmtAbs.MulInt64(dir.ToMultiplier()))
	if err != nil {
		return market, sdk.Dec{}, err
	}

	quoteDelta := quoteReserveAbs.Neg().MulInt64(dir.ToMultiplier())

	if err := market.HasEnoughReservesForTrade(quoteReserveAbs, baseAmtAbs); err != nil {
		return market, sdk.Dec{}, err
	}
	quoteAssetAmtAbs = market.FromQuoteReserveToAsset(quoteReserveAbs)

	if err := checkIfLimitIsViolated(quoteLimit, quoteAssetAmtAbs, dir); err != nil {
		return market, sdk.Dec{}, err
	}

	baseAmt = baseAmtAbs.MulInt64(dir.ToMultiplier())

	updatedMarket, err = k.executeSwap(ctx, market, quoteDelta, baseAmt, skipFluctuationLimitCheck)
	if err != nil {
		return market, sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	return updatedMarket, quoteAssetAmtAbs, err
}

func (k Keeper) executeSwap(
	ctx sdk.Context, market types.Market, quoteDelta sdk.Dec, baseDelta sdk.Dec,
	skipFluctuationLimitCheck bool,
) (newMarket types.Market, err error) {
	// -------------------- Update reserves
	market.AddToBaseReserveAndTotalLongShort(baseDelta)
	market.AddToQuoteReserve(quoteDelta)

	if err = k.updatePool(ctx, market, skipFluctuationLimitCheck); err != nil {
		return newMarket, fmt.Errorf("error updating reserve: %w", err)
	}

	// -------------------- Emit events
	if err = ctx.EventManager().EmitTypedEvent(&types.MarkPriceChangedEvent{
		Pair:           market.Pair,
		Price:          market.GetMarkPrice(),
		BlockTimestamp: ctx.BlockTime(),
	}); err != nil {
		return newMarket, err
	}

	if err = ctx.EventManager().EmitTypedEvent(&types.SwapEvent{
		Pair:        market.Pair,
		QuoteAmount: quoteDelta,
		BaseAmount:  baseDelta,
	}); err != nil {
		return newMarket, err
	}

	newMarket = market
	return newMarket, err
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
	market types.Market,
	dir types.Direction,
	quoteAmt sdk.Dec,
	baseLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedMarket types.Market, baseAmtAbs sdk.Dec, err error) {
	if quoteAmt.IsZero() {
		return types.Market{}, sdk.ZeroDec(), nil
	}

	if _, err = k.oracleKeeper.GetExchangeRate(ctx, market.Pair); err != nil {
		return types.Market{}, sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", market.Pair)
	}

	// check trade limit ratio on quote in either direction
	quoteResrveAbs := market.FromQuoteAssetToReserve(quoteAmt).Abs()
	baseAmtAbs, err = market.GetBaseAmountByQuoteAmount(
		quoteResrveAbs.MulInt64(dir.ToMultiplier()))
	if err != nil {
		return types.Market{}, sdk.Dec{}, err
	}

	if err := market.HasEnoughReservesForTrade(quoteResrveAbs, baseAmtAbs); err != nil {
		return types.Market{}, sdk.Dec{}, err
	}

	if err := checkIfLimitIsViolated(baseLimit, baseAmtAbs, dir); err != nil {
		return types.Market{}, sdk.Dec{}, err
	}

	quoteAmt = quoteResrveAbs.MulInt64(dir.ToMultiplier())
	baseDelta := baseAmtAbs.Neg().MulInt64(dir.ToMultiplier())

	updatedMarket, err = k.executeSwap(ctx, market, quoteAmt, baseDelta, skipFluctuationLimitCheck)
	if err != nil {
		return types.Market{}, sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	return updatedMarket, baseAmtAbs, err
}

// checkIfLimitIsViolated checks if the limit is violated by the amount.
// returns error if it does
func checkIfLimitIsViolated(limit, amount sdk.Dec, dir types.Direction) error {
	if !limit.IsZero() {
		if dir == types.Direction_LONG && amount.LT(limit) {
			return types.ErrAssetFailsUserLimit.Wrapf(
				"amount (%s) is less than selected limit (%s)",
				amount.String(),
				limit.String(),
			)
		} else if dir == types.Direction_SHORT && amount.GT(limit) {
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
This should run prior to updating the snapshot, otherwise it will compare the currently updated market to itself.

args:
  - ctx: the cosmos-sdk context
  - pool: the updated market

ret:
  - err: error if any
*/
func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool types.Market) error {
	if pool.Config.FluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	latestSnapshot, err := k.GetLastSnapshot(ctx, pool)
	if err != nil {
		return err
	}
	if pool.IsOverFluctuationLimitInRelationWithSnapshot(latestSnapshot) {
		return types.ErrOverFluctuationLimit
	}

	return nil
}

/*
GetLastSnapshot retrieve the last snapshot for a particular market

args:
  - ctx: the cosmos-sdk context
  - pool: the market to check
*/
func (k Keeper) GetLastSnapshot(ctx sdk.Context, pool types.Market) (types.ReserveSnapshot, error) {
	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}.Prefix(pool.Pair).Descending())
	defer it.Close()
	if !it.Valid() {
		return types.ReserveSnapshot{}, fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
	}
	latestSnapshot := it.Value()
	return latestSnapshot, nil
}

/*
IsOverSpreadLimit compares the current spot price of the market (given by pair) to the underlying's index price (given by an oracle).
It panics if you provide it with a pair that doesn't exist in the state.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - bool: whether the price has deviated from the oracle price beyond a spread ratio
*/
func (k Keeper) IsOverSpreadLimit(ctx sdk.Context, pair asset.Pair) (bool, error) {
	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return false, err
	}

	indexPrice, err := k.oracleKeeper.GetExchangeRate(ctx, pair)
	if err != nil {
		return false, err
	}

	return pool.IsOverSpreadLimit(indexPrice), nil
}

/*
GetMaintenanceMarginRatio returns the maintenance margin ratio for the pool from the asset pair.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair

ret:
  - sdk.Dec: The maintenance margin ratio for the pool
  - error
*/
func (k Keeper) GetMaintenanceMarginRatio(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error) {
	pool, err := k.Pools.Get(ctx, pair)
	if err != nil {
		return sdk.Dec{}, err
	}

	return pool.Config.MaintenanceMarginRatio, nil
}

/*
GetAllPools returns an array of all the pools

args:
  - ctx: the cosmos-sdk context

ret:
  - []types.Market: All defined market
*/
func (k Keeper) GetAllPools(ctx sdk.Context) []types.Market {
	return k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
}

func (k Keeper) GetPool(ctx sdk.Context, pair asset.Pair) (types.Market, error) {
	return k.Pools.Get(ctx, pair)
}
