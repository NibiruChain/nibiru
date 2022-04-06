package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
Takes a pool and the amount of tokens desired to add to the pool,
and calculates the number of pool shares and remaining coins after theoretically
adding the tokensIn to the pool.

Note that this function is pure/read-only. It only calculates the theoretical amoount
and doesn't modify the actual state.

args:
  - pool: the pool containing assets
  - tokensIn: a slice of coins to add to the pool

ret:
  - numShares: the number of LP shares representing the maximal number of tokens added to the pool
  - remCoins: the remaining number of coins after adding the tokens

*/
func (pool Pool) maximalSharesFromExactRatioJoin(tokensIn sdk.Coins) (numShares sdk.Int, remCoins sdk.Coins, err error) {
	coinShareRatios := make([]sdk.Dec, len(tokensIn))
	minShareRatio := sdk.MaxSortableDec
	maxShareRatio := sdk.ZeroDec()

	poolLiquidity := GetPoolLiquidity(pool.PoolAssets)

	for i, coin := range tokensIn {
		shareRatio := coin.Amount.ToDec().QuoInt(poolLiquidity.AmountOfNoDenomValidation(coin.Denom))
		if shareRatio.LT(minShareRatio) {
			minShareRatio = shareRatio
		}
		if shareRatio.GT(maxShareRatio) {
			maxShareRatio = shareRatio
		}
		coinShareRatios[i] = shareRatio
	}

	if minShareRatio.Equal(sdk.MaxSortableDec) {
		return sdk.ZeroInt(), sdk.NewCoins(), errors.New("unexpected error in balancer maximalExactRatioJoin")
	}

	if minShareRatio.IsZero() {
		return sdk.ZeroInt(), tokensIn, nil
	}

	numShares = minShareRatio.MulInt(pool.TotalShares.Amount).TruncateInt()
	remCoins = sdk.Coins{}

	// if we have multiple shares, calculate remCoins
	if !minShareRatio.Equal(maxShareRatio) {
		// we have to calculate remCoins
		for i, coin := range tokensIn {
			if !coinShareRatios[i].Equal(minShareRatio) {
				usedAmount := minShareRatio.MulInt(
					poolLiquidity.AmountOfNoDenomValidation(coin.Denom)).Ceil().TruncateInt()
				remainingAmount := coin.Amount.Sub(usedAmount)
				// add to RemCoins
				if !remainingAmount.IsZero() {
					remCoins = remCoins.Add(sdk.Coin{Denom: coin.Denom, Amount: remainingAmount})
				}
			}
		}
	}

	return numShares, remCoins, nil
}

/*
Adds new liquidity to the pool and increments the total number of shares.

args:
  - numShares: the number of LP shares to increment
  - newLiquidity: the new tokens to deposit into the pool
*/
func (pool *Pool) updateLiquidity(numShares sdk.Int, newLiquidity sdk.Coins) (err error) {
	for _, coin := range newLiquidity {
		i, poolAsset, err := getPoolAssetAndIndex(pool.PoolAssets, coin.Denom)
		if err != nil {
			return err
		}
		poolAsset.Token.Amount = poolAsset.Token.Amount.Add(coin.Amount)
		pool.PoolAssets[i] = poolAsset
	}
	pool.TotalShares.Amount = pool.TotalShares.Amount.Add(numShares)
	return nil
}
