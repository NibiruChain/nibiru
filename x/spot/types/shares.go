package types

import (
	"errors"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
Takes a pool and the amount of tokens desired to add to the pool,
and calculates the number of pool shares and remaining coins after theoretically
adding the tokensIn to the pool.

Note that this function is pure/read-only. It only calculates the theoretical amoount
and doesn't modify the actual state.

args:
  - tokensIn: a slice of coins to add to the pool

ret:
  - numShares: the number of LP shares representing the maximal number of tokens added to the pool
  - remCoins: the remaining number of coins after adding the tokens
  - err: error if any
*/
func (pool Pool) numSharesOutFromTokensIn(tokensIn sdk.Coins) (
	numShares sdk.Int, remCoins sdk.Coins, err error,
) {
	coinShareRatios := make([]sdk.Dec, len(tokensIn))
	minShareRatio := sdk.MaxSortableDec
	maxShareRatio := sdk.ZeroDec()

	poolLiquidity := pool.PoolBalances()
	if len(tokensIn) == 1 {
		// From balancer whitepaper, for 2 assets with the same weight, the shares issued are:
		// P_{supply} * (sqrt(1+((1-f/2) * x_{in})/X)-1)

		one := sdk.OneDec()

		joinShare := sdk.NewDecFromInt(tokensIn[0].Amount).Mul(one.Sub(pool.PoolParams.SwapFee.Quo(sdk.NewDec(2)))).QuoInt(
			poolLiquidity.AmountOfNoDenomValidation(tokensIn[0].Denom),
		).Add(one)

		joinShare, err = joinShare.ApproxSqrt()
		if err != nil {
			return
		}

		numShares = joinShare.Sub(one).MulInt(pool.TotalShares.Amount).TruncateInt()
		return
	}

	for i, coin := range tokensIn {
		shareRatio := sdk.NewDecFromInt(coin.Amount).QuoInt(
			poolLiquidity.AmountOfNoDenomValidation(coin.Denom),
		)
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
For a stableswap pool, takes the amount of tokens desired to add to the pool,
and calculates the number of pool shares and remaining coins after theoretically
adding the tokensIn to the pool. All tokens are used in this function.

The delta in number of share follows the evolution of the constant of the pool. E.g. if someone bring tokens
to increase the value D of the pool by 10%, he will receive 10% of the existing token share.

Note that this function is pure/read-only. It only calculates the theoretical amoount
and doesn't modify the actual state.

args:
  - tokensIn: a slice of coins to add to the pool

ret:
  - numShares: the number of LP shares representing the maximal number of tokens added to the pool
  - remCoins: the remaining number of coins after adding the tokens
  - err: error if any
*/
func (pool Pool) numSharesOutFromTokensInStableSwap(tokensIn sdk.Coins) (
	numShares sdk.Int, err error,
) {
	tokenSupply := pool.TotalShares.Amount

	D, err := pool.GetD(pool.PoolAssets)
	if err != nil {
		return
	}
	D0 := sdk.NewInt(int64(D.Uint64()))

	var newPoolAssets []PoolAsset

	for assetIndex, poolAsset := range pool.PoolAssets {
		inAmount := tokensIn.AmountOf(poolAsset.Token.Denom)

		if !inAmount.IsZero() {
			newAmount := pool.PoolAssets[assetIndex].Token.Amount.Add(inAmount)

			newPoolAssets = append(newPoolAssets, PoolAsset{Token: sdk.NewCoin(poolAsset.Token.Denom, newAmount)})
		} else {
			newPoolAssets = append(newPoolAssets, poolAsset)
		}
	}

	newD, err := pool.GetD(newPoolAssets)
	if err != nil {
		return
	}
	D1 := sdk.NewInt(int64(newD.Uint64()))
	if D1.LT(D0) {
		// Should not happen
		err = ErrInvariantLowerAfterJoining
		return
	}

	// Calculate, how much pool tokens to mint
	numShares = tokenSupply.Mul(D1.Sub(D0)).Quo(D0)

	return
}

/*
TokensOutFromPoolSharesIn Calculates the number of tokens to remove from liquidity given LP shares returned to the pool.

Note that this function is pure/read-only. It only calculates the theoretical amoount
and doesn't modify the actual state.

args:
  - numSharesIn: number of LP shares to return to the pool

ret:
  - tokensOut: the tokens withdrawn from the pool
  - fees: the fees collected
  - err: error if any
*/
func (pool Pool) TokensOutFromPoolSharesIn(numSharesIn sdk.Int) (
	tokensOut sdk.Coins, fees sdk.Coins, err error,
) {
	if numSharesIn.IsZero() {
		return nil, nil, errors.New("num shares in must be greater than zero")
	}

	shareRatio := sdk.NewDecFromInt(numSharesIn).QuoInt(pool.TotalShares.Amount)
	if shareRatio.IsZero() {
		return nil, nil, errors.New("share ratio must be greater than zero")
	}
	if shareRatio.GT(sdk.OneDec()) {
		return nil, nil, errors.New("share ratio cannot be greater than one")
	}

	poolLiquidity := pool.PoolBalances()
	tokensOut = make(sdk.Coins, len(poolLiquidity))
	fees = make(sdk.Coins, len(poolLiquidity))
	for i, coin := range poolLiquidity {
		// tokenOut = shareRatio * poolTokenAmt * (1 - exitFee)
		tokenAmount := shareRatio.MulInt(coin.Amount)
		tokenOutAmt := tokenAmount.Mul(
			sdk.OneDec().Sub(pool.PoolParams.ExitFee),
		).TruncateInt()
		tokensOut[i] = sdk.NewCoin(coin.Denom, tokenOutAmt)
		fees[i] = sdk.NewCoin(coin.Denom, tokenAmount.TruncateInt().Sub(tokenOutAmt))
	}

	return tokensOut, sdk.NewCoins(fees...), nil
}

/*
Compute the minimum number of shares a user need to provide to get at least one u-token
*/
func (pool Pool) MinSharesInForTokensOut() (minShares sdk.Int) {
	poolLiquidity := pool.PoolBalances()

	minShares = sdk.ZeroInt()

	for _, coin := range poolLiquidity {
		shareRatio := sdk.MustNewDecFromStr("2").Quo(sdk.NewDecFromInt(coin.Amount).Quo(sdk.OneDec().Sub(pool.PoolParams.ExitFee)))

		shares := shareRatio.MulInt(pool.TotalShares.Amount).TruncateInt()

		if minShares.IsZero() || minShares.LT(shares) {
			minShares = shares
		}
	}
	return
}

/*
Adds new liquidity to the pool and increments the total number of shares.

args:
  - numShares: the number of LP shares to increment
  - newLiquidity: the new tokens to deposit into the pool
*/
func (pool *Pool) incrementBalances(numShares sdkmath.Int, newLiquidity sdk.Coins) (
	err error,
) {
	for _, coin := range newLiquidity {
		i, poolAsset, err := pool.getPoolAssetAndIndex(coin.Denom)
		if err != nil {
			return err
		}
		poolAsset.Token.Amount = poolAsset.Token.Amount.Add(coin.Amount)
		pool.PoolAssets[i] = poolAsset
	}
	pool.TotalShares.Amount = pool.TotalShares.Amount.Add(numShares)
	return nil
}
