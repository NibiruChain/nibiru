package keeper

// Everything to do with total liquidity in the spot and liquidity of specific coin denoms.

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"

	spottypes "github.com/NibiruChain/nibiru/x/spot/types"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

/*
Fetches the liquidity for a specific coin denom.

args:

	ctx: the cosmos-sdk context
	denom: the coin denom

ret:

	amount: the amount of liquidity for the provided coin. Returns 0 if not found.
*/
func (k Keeper) GetDenomLiquidity(ctx sdk.Context, denom string) (amount sdkmath.Int, err error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(spottypes.GetDenomLiquidityPrefix(denom))
	if bz == nil {
		return sdk.NewInt(0), nil
	}

	if err := amount.Unmarshal(bz); err != nil {
		return amount, common.CombineErrors(
			fmt.Errorf("failed to GetDenomLiquidty for denom %s", denom),
			err)
	}

	return amount, nil
}

/*
Sets the liquidity for a specific coin denom.

args:

	ctx: the cosmos-sdk context
	denom: the coin denom
	amount: the amount of liquidity for the coin
*/
func (k Keeper) SetDenomLiquidity(ctx sdk.Context, denom string, amount sdkmath.Int) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := amount.Marshal()
	if err != nil {
		return err
	}
	store.Set(spottypes.GetDenomLiquidityPrefix(denom), bz)
	return nil
}

/*
Fetches the liquidity for all tokens in the spot.

args:

	ctx: the cosmos-sdk context

ret:

	coins: an array of liquidities in the spot
*/
func (k Keeper) GetTotalLiquidity(ctx sdk.Context) (coins sdk.Coins) {
	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, spottypes.KeyTotalLiquidity)

	iterator := prefixStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Key())
		amount, err := k.GetDenomLiquidity(ctx, denom)
		if err != nil {
			ctx.Logger().Error(err.Error())
		} else {
			coins = coins.Add(sdk.NewCoin(denom, amount))
		}
	}

	return coins
}

/*
Sets the total liquidity for each coin.

args:

	ctx: the cosmos-sdk context
	coins: the array of liquidities to update with
*/
func (k Keeper) SetTotalLiquidity(ctx sdk.Context, coins sdk.Coins) error {
	for _, coin := range coins {
		if err := k.SetDenomLiquidity(ctx, coin.Denom, coin.Amount); err != nil {
			return err
		}
	}
	return nil
}

/*
Increases the total liquidity of the provided coins by the coin amount.

args:

	ctx: the cosmos-sdk context
	coins: the coins added to the spot
*/
func (k Keeper) RecordTotalLiquidityIncrease(ctx sdk.Context, coins sdk.Coins) error {
	for _, coin := range coins {
		amount, err := k.GetDenomLiquidity(ctx, coin.Denom)
		if err != nil {
			return err
		}
		amount = amount.Add(coin.Amount)
		if err := k.SetDenomLiquidity(ctx, coin.Denom, amount); err != nil {
			return err
		}
	}
	return nil
}

/*
Increases the total liquidity of the provided coins by the coin amount.

args:

	ctx: the cosmos-sdk context
	coins: the coins removed from the spot
*/
func (k Keeper) RecordTotalLiquidityDecrease(ctx sdk.Context, coins sdk.Coins) error {
	for _, coin := range coins {
		amount, err := k.GetDenomLiquidity(ctx, coin.Denom)
		if err != nil {
			return err
		}
		amount = amount.Sub(coin.Amount)
		if err := k.SetDenomLiquidity(ctx, coin.Denom, amount); err != nil {
			return err
		}
	}
	return nil
}
