package types

import (
	"errors"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/x/common"
)

/*
For a single asset join, compute the number of token that need to be swapped for an optimal swap and join.
See https://www.notion.so/nibiru/Single-Asset-Join-Math-2075178cb9684062b9b65ad23ff14417

args:
  - tokenIn: the token to add to the pool

ret:
  - out: the tokens to swap before joining the pool
  - err: error if any
*/
func (pool *Pool) SwapForSwapAndJoin(tokenIn sdk.Coin) (
	out sdk.Coin, err error,
) {
	PRECISION := int64(1_000_000)

	mu := sdk.OneDec().Quo(sdk.OneDec().Sub(pool.PoolParams.SwapFee))
	sigma := (sdk.OneDec().Add(mu))

	mu = mu.MulInt64(PRECISION).MulInt64(PRECISION)
	sigma = sigma.MulInt64(PRECISION)

	lx := pool.PoolBalances().AmountOfNoDenomValidation(tokenIn.Denom).ToDec()

	lxsigmauint256 := uint256.NewInt()
	lxsigmauint256.SetFromBig(lx.Mul(sigma).BigInt())

	lxmuuint256 := uint256.NewInt()
	lxmuuint256.SetFromBig(lx.Mul(mu).BigInt())

	xinuint256 := uint256.NewInt()
	xinuint256.SetFromBig(tokenIn.Amount.ToDec().BigInt())

	squarable := uint256.NewInt().Add(
		uint256.NewInt().Mul(
			lxsigmauint256,
			lxsigmauint256,
		),
		uint256.NewInt().Mul(
			uint256.NewInt().SetUint64(4),
			uint256.NewInt().Mul(
				lxmuuint256,
				xinuint256,
			),
		),
	)

	BigInt := &big.Int{}
	sqrt := BigInt.Quo(BigInt.Sqrt(squarable.ToBig()), big.NewInt(PRECISION))

	sqrtFactor := sdk.NewDecFromBigIntWithPrec(sqrt, int64(common.BigIntPrecision))

	amount := (sigma.Mul(lx).MulInt(sdk.NewInt(-1)).QuoInt64(PRECISION).Add(sqrtFactor)).Quo(sdk.MustNewDecFromStr("2"))
	return sdk.NewCoin(tokenIn.Denom, amount.TruncateInt()), err
}

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

	for i, coin := range tokensIn {
		shareRatio := coin.Amount.ToDec().QuoInt(
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

	D0 := sdk.NewInt(int64(pool.getD(pool.PoolAssets).Uint64()))

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

	D1 := sdk.NewInt(int64(pool.getD(newPoolAssets).Uint64()))
	if D1.LT(D0) {
		// Should not happen
		panic(nil)
	}

	// Calculate, how much pool tokens to mint
	numShares = tokenSupply.Mul(D1.Sub(D0)).Quo(D0)

	return
}

/*
Calculates the number of tokens to remove from liquidity given LP shares returned to the pool.

Note that this function is pure/read-only. It only calculates the theoretical amoount
and doesn't modify the actual state.

args:
  - numSharesIn: number of LP shares to return to the pool

ret:
  - tokensOut: the tokens withdrawn from the pool
  - err: error if any
*/
func (pool Pool) TokensOutFromPoolSharesIn(numSharesIn sdk.Int) (
	tokensOut sdk.Coins, err error,
) {
	if numSharesIn.IsZero() {
		return nil, errors.New("num shares in must be greater than zero")
	}

	shareRatio := numSharesIn.ToDec().QuoInt(pool.TotalShares.Amount)
	if shareRatio.IsZero() {
		return nil, errors.New("share ratio must be greater than zero")
	}
	if shareRatio.GT(sdk.OneDec()) {
		return nil, errors.New("share ratio cannot be greater than one")
	}

	poolLiquidity := pool.PoolBalances()
	tokensOut = make(sdk.Coins, len(poolLiquidity))
	for i, coin := range poolLiquidity {
		// tokenOut = shareRatio * poolTokenAmt * (1 - exitFee)
		tokenOutAmt := shareRatio.MulInt(coin.Amount).Mul(
			sdk.OneDec().Sub(pool.PoolParams.ExitFee),
		).TruncateInt()
		tokensOut[i] = sdk.NewCoin(coin.Denom, tokenOutAmt)
	}

	return tokensOut, nil
}

/*
Adds new liquidity to the pool and increments the total number of shares.

args:
  - numShares: the number of LP shares to increment
  - newLiquidity: the new tokens to deposit into the pool
*/
func (pool *Pool) incrementBalances(numShares sdk.Int, newLiquidity sdk.Coins) (
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
