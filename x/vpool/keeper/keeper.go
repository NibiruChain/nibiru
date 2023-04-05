package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/vpool/types"
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
		Pools:        collections.NewMap(storeKey, 0, asset.PairKeyEncoder, collections.ProtoValueEncoder[types.Vpool](codec)),
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

	Pools            collections.Map[asset.Pair, types.Vpool]
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
	vpool types.Vpool,
	dir types.Direction,
	baseAmt sdk.Dec,
	quoteLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedVpool types.Vpool, quoteAmtAbs sdk.Dec, err error) {
	if baseAmt.IsZero() {
		return vpool, sdk.ZeroDec(), nil
	}

	if _, err = k.oracleKeeper.GetExchangeRate(ctx, vpool.Pair); err != nil {
		return vpool, sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", vpool.Pair)
	}

	baseAmtAbs := baseAmt.Abs()
	quoteAmtAbs, err = vpool.GetQuoteAmountByBaseAmount(baseAmtAbs.MulInt64(dir.ToMultiplier()))
	if err != nil {
		return vpool, sdk.Dec{}, err
	}

	if err := vpool.HasEnoughReservesForTrade(quoteAmtAbs, baseAmtAbs); err != nil {
		return vpool, sdk.Dec{}, err
	}

	if err := checkIfLimitIsViolated(quoteLimit, quoteAmtAbs, dir); err != nil {
		return vpool, sdk.Dec{}, err
	}

	quoteDelta := quoteAmtAbs.Neg().MulInt64(dir.ToMultiplier())
	baseAmt = baseAmtAbs.MulInt64(dir.ToMultiplier())

	vpool.Bias = vpool.Bias.Add(baseAmt.Neg())

	updatedVpool, err = k.executeSwap(ctx, vpool, quoteDelta, baseAmt, skipFluctuationLimitCheck)
	if err != nil {
		return vpool, sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	return updatedVpool, quoteAmtAbs, err
}

func (k Keeper) executeSwap(
	ctx sdk.Context, vpool types.Vpool, quoteDelta sdk.Dec, baseDelta sdk.Dec,
	skipFluctuationLimitCheck bool,
) (newVpool types.Vpool, err error) {
	// -------------------- Update reserves
	vpool.AddToBaseAssetReserve(baseDelta)
	vpool.AddToQuoteAssetReserve(quoteDelta)

	if err = k.updatePool(ctx, vpool, skipFluctuationLimitCheck); err != nil {
		return newVpool, fmt.Errorf("error updating reserve: %w", err)
	}

	// -------------------- Emit events
	if err = ctx.EventManager().EmitTypedEvent(&types.MarkPriceChangedEvent{
		Pair:           vpool.Pair,
		Price:          vpool.GetMarkPrice(),
		BlockTimestamp: ctx.BlockTime(),
	}); err != nil {
		return newVpool, err
	}

	if err = ctx.EventManager().EmitTypedEvent(&types.SwapOnVpoolEvent{
		Pair:        vpool.Pair,
		QuoteAmount: quoteDelta,
		BaseAmount:  baseDelta,
	}); err != nil {
		return newVpool, err
	}

	newVpool = vpool
	return newVpool, err
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
	vpool types.Vpool,
	dir types.Direction,
	quoteAmt sdk.Dec,
	baseLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedVpool types.Vpool, baseAmtAbs sdk.Dec, err error) {
	if quoteAmt.IsZero() {
		return types.Vpool{}, sdk.ZeroDec(), nil
	}

	if _, err = k.oracleKeeper.GetExchangeRate(ctx, vpool.Pair); err != nil {
		return types.Vpool{}, sdk.Dec{}, types.ErrNoValidPrice.Wrapf("%s", vpool.Pair)
	}

	// check trade limit ratio on quote in either direction
	quoteAmtAbs := quoteAmt.Abs()
	baseAmtAbs, err = vpool.GetBaseAmountByQuoteAmount(
		quoteAmtAbs.MulInt64(dir.ToMultiplier()))
	if err != nil {
		return types.Vpool{}, sdk.Dec{}, err
	}

	if err := vpool.HasEnoughReservesForTrade(quoteAmtAbs, baseAmtAbs); err != nil {
		return types.Vpool{}, sdk.Dec{}, err
	}

	if err := checkIfLimitIsViolated(baseLimit, baseAmtAbs, dir); err != nil {
		return types.Vpool{}, sdk.Dec{}, err
	}

	quoteAmt = quoteAmtAbs.MulInt64(dir.ToMultiplier())
	baseDelta := baseAmtAbs.Neg().MulInt64(dir.ToMultiplier())

	vpool.Bias = vpool.Bias.Add(baseAmtAbs.MulInt64(dir.ToMultiplier()))

	updatedVpool, err = k.executeSwap(ctx, vpool, quoteAmt, baseDelta, skipFluctuationLimitCheck)
	if err != nil {
		return types.Vpool{}, sdk.Dec{}, fmt.Errorf("error updating reserve: %w", err)
	}

	return updatedVpool, baseAmtAbs, err
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
func (k Keeper) checkFluctuationLimitRatio(ctx sdk.Context, pool types.Vpool) error {
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
GetLastSnapshot retrieve the last snapshot for a particular vpool

args:
  - ctx: the cosmos-sdk context
  - pool: the vpool to check
*/
func (k Keeper) GetLastSnapshot(ctx sdk.Context, pool types.Vpool) (types.ReserveSnapshot, error) {
	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}.Prefix(pool.Pair).Descending())
	defer it.Close()
	if !it.Valid() {
		return types.ReserveSnapshot{}, fmt.Errorf("error getting last snapshot number for pair %s", pool.Pair)
	}
	latestSnapshot := it.Value()
	return latestSnapshot, nil
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
  - []types.Vpool: All defined vpool
*/
func (k Keeper) GetAllPools(ctx sdk.Context) []types.Vpool {
	return k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
}

func (k Keeper) GetPool(ctx sdk.Context, pair asset.Pair) (types.Vpool, error) {
	return k.Pools.Get(ctx, pair)
}
