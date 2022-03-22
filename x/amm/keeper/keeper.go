package keeper

import (
	"github.com/MatrixDao/matrix/x/amm/types"
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

// SwapInput swaps pair token
func (k Keeper) SwapInput(ctx sdk.Context, pair string, dir types.Direction, quoteAssetAmount sdk.Int) (sdk.Int, error) {
	if !k.ExistsPool(ctx, pair) {
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

	//amount := types.GetBaseAmountByQuoteAmount(dir, pool, quoteAssetAmount)

	return sdk.NewInt(1234), nil
}

// getBaseAmountByQuoteAmount returns the amount that you will get by specific quote amount
func getBaseAmountByQuoteAmount(dir types.Direction, pair string, quoteAmount sdk.Int) sdk.Int {
	return sdk.ZeroInt()
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
	tradeLimitRatio sdk.Int, // integer with 6 decimals, 1_000_000 means 1.0
	quoteAssetReserve sdk.Int,
	baseAssetReserve sdk.Int,
) error {
	store := ctx.KVStore(k.storeKey)

	pool := &types.Pool{
		Pair:              pair,
		TradeLimitRatio:   tradeLimitRatio.String(),
		QuoteAssetReserve: quoteAssetReserve.String(),
		BaseAssetReserve:  baseAssetReserve.String(),
	}

	bz, err := k.codec.Marshal(pool)
	if err != nil {
		return err
	}

	store.Set(types.GetPoolKey(pool.Pair), bz)

	return nil
}

// ExistsPool returns true if pool exists, false if not.
func (k Keeper) ExistsPool(ctx sdk.Context, pair string) bool {
	store := k.getStore(ctx)
	return store.Has(types.GetPoolKey(pair))
}

func (k Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}
