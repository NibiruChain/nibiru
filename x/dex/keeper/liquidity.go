package keeper

// Everything to do with total liquidity in the dex and liquidity of specific coin denoms.

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/dex/types"
)

/*
Fetches the liquidity for a specific coin denom.

args:
  ctx: the cosmos-sdk context
  denom: the coin denom

ret:
  amount: the amount of liquidity for the provided coin. Returns 0 if not found.
*/
func (k Keeper) GetDenomLiquidity(ctx sdk.Context, denom string) (amount sdk.Int) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetDenomLiquidityPrefix(denom))
	if bz == nil {
		return sdk.NewInt(0)
	}

	if err := amount.Unmarshal(bz); err != nil {
		panic(err)
	}

	return amount
}

/*
Sets the liquidity for a specific coin denom.

args:
  ctx: the cosmos-sdk context
  denom: the coin denom
  amount: the amount of liquidity for the coin
*/
func (k Keeper) SetDenomLiquidity(ctx sdk.Context, denom string, amount sdk.Int) {
	store := ctx.KVStore(k.storeKey)
	bz, err := amount.Marshal()
	if err != nil {
		panic(err)
	}
	store.Set(types.GetDenomLiquidityPrefix(denom), bz)
}

/*
Fetches the liquidity for all tokens in the dex.

args:
  ctx: the cosmos-sdk context

ret:
  coins: an array of liquidities in the dex
*/
func (k Keeper) GetTotalLiquidity(ctx sdk.Context) (coins sdk.Coins) {
	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, types.KeyTotalLiquidity)

	iterator := prefixStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Key())
		amount := k.GetDenomLiquidity(ctx, denom)
		coins = coins.Add(sdk.NewCoin(denom, amount))
	}

	return coins
}

/*
Sets the total liquidity for each coin.

args:
  ctx: the cosmos-sdk context
  coins: the array of liquidities to update with
*/
func (k Keeper) SetTotalLiquidity(ctx sdk.Context, coins sdk.Coins) {
	for _, coin := range coins {
		k.SetDenomLiquidity(ctx, coin.Denom, coin.Amount)
	}
}

/*
Increases the total liquidity of the provided coins by the coin amount.

args:
  ctx: the cosmos-sdk context
  coins: the coins added to the dex
*/
func (k Keeper) RecordTotalLiquidityIncrease(ctx sdk.Context, coins sdk.Coins) {
	for _, coin := range coins {
		amount := k.GetDenomLiquidity(ctx, coin.Denom)
		amount = amount.Add(coin.Amount)
		k.SetDenomLiquidity(ctx, coin.Denom, amount)
	}
}

/*
Increases the total liquidity of the provided coins by the coin amount.

args:
  ctx: the cosmos-sdk context
  coins: the coins removed from the dex
*/
func (k Keeper) RecordTotalLiquidityDecrease(ctx sdk.Context, coins sdk.Coins) {
	for _, coin := range coins {
		amount := k.GetDenomLiquidity(ctx, coin.Denom)
		amount = amount.Sub(coin.Amount)
		k.SetDenomLiquidity(ctx, coin.Denom, amount)
	}
}
