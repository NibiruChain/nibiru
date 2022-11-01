package types

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMaximalSharesFromExactRatioJoin(t *testing.T) {
	for _, tc := range []struct {
		name              string
		poolAssets        []PoolAsset
		existingShares    int64
		tokensIn          sdk.Coins
		expectedNumShares sdk.Int
		expectedRemCoins  sdk.Coins
	}{
		{
			name: "all coins deposited",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedNumShares: sdk.NewInt(100),
			expectedRemCoins:  sdk.NewCoins(),
		},
		{
			name: "some coins deposited",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 50),
			),
			expectedNumShares: sdk.NewInt(50),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
			),
		},
		{
			name: "limited by smallest amount",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1),
				sdk.NewInt64Coin("bbb", 50),
			),
			expectedNumShares: sdk.NewInt(1),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 49),
			),
		},
		{
			name: "limited by smallest amount - 2",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 200),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 10),
			),
			expectedNumShares: sdk.NewInt(5),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 5),
			),
		},
		{
			name: "right number of LP shares",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 50),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 100),
				},
			},
			existingShares: 150,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 50),
			),
			expectedNumShares: sdk.NewInt(75),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 25),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := Pool{
				Id:          1,
				Address:     "some_address",
				PoolParams:  PoolParams{},
				PoolAssets:  tc.poolAssets,
				TotalWeight: sdk.OneInt(),
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", tc.existingShares),
			}
			numShares, remCoins, _ := pool.numSharesOutFromTokensIn(tc.tokensIn)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
		})
	}
}

func TestSwapForSwapAndJoin(t *testing.T) {
	for _, tc := range []struct {
		name             string
		poolAssets       []PoolAsset
		existingShares   int64
		tokensIn         sdk.Coins
		expectedX0Denom  string
		expectedX0Amount int64
		err              error
	}{
		{
			name: "tokens bbb need to be swapped, but no swap necessary",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 101),
			),
			expectedX0Denom:  "bbb",
			expectedX0Amount: 0,
		},
		{
			name: "tokens aaa need to be swapped, but no swap necessary",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 101),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedX0Denom:  "aaa",
			expectedX0Amount: 0,
		},
		{
			name: "tokens aaa need to be swapped",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 101),
				sdk.NewInt64Coin("bbb", 43),
			),
			expectedX0Denom:  "aaa",
			expectedX0Amount: 27,
		},
		{
			name: "tokens bbb need to be swapped",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 43),
				sdk.NewInt64Coin("bbb", 101),
			),
			expectedX0Denom:  "bbb",
			expectedX0Amount: 27,
		},
		{
			name: "single asset join, aaa",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 230),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 928),
			),
			expectedX0Denom:  "aaa",
			expectedX0Amount: 286,
		},
		{
			name: "single asset join, bbb",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 230),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 928),
			),
			expectedX0Denom:  "bbb",
			expectedX0Amount: 388,
		},

		{
			name: "3 asset pool, raise issue",
			poolAssets: []PoolAsset{
				{
					Token:  sdk.NewInt64Coin("aaa", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("bbb", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
				{
					Token:  sdk.NewInt64Coin("ccc", 1000),
					Weight: sdk.NewInt(1 << 30),
				},
			},
			existingShares: 100,
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 43),
				sdk.NewInt64Coin("bbb", 101),
			),
			err: errors.New("swap and add tokens to pool only available for 2 assets pool"),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := Pool{
				Id:          1,
				Address:     "some_address",
				PoolParams:  PoolParams{SwapFee: sdk.ZeroDec()},
				PoolAssets:  tc.poolAssets,
				TotalWeight: sdk.NewInt(2 << 30),
				TotalShares: sdk.NewCoin("nibiru/pool/1", sdk.NewIntWithDecimal(100, 18)),
			}
			swapCoin, err := pool.SwapForSwapAndJoin(tc.tokensIn)
			if tc.err == nil {
				require.NoError(t, err)
				require.EqualValues(t, tc.expectedX0Denom, swapCoin.Denom)
				require.EqualValues(t, tc.expectedX0Amount, swapCoin.Amount.Int64())

				if swapCoin.Amount.GT(sdk.ZeroInt()) {
					index, _, err := pool.getPoolAssetAndIndex(swapCoin.Denom)

					require.NoError(t, err)
					otherDenom := pool.PoolAssets[1-index].Token.Denom

					tokenOut, err := pool.CalcOutAmtGivenIn(swapCoin, otherDenom, false)
					require.NoError(t, err)

					err = pool.ApplySwap(swapCoin, tokenOut)
					require.NoError(t, err)

					tokensIn := sdk.Coins{
						{
							Denom:  swapCoin.Denom,
							Amount: tc.tokensIn.AmountOfNoDenomValidation(swapCoin.Denom).Sub(swapCoin.Amount),
						},
						{
							Denom:  otherDenom,
							Amount: tc.tokensIn.AmountOfNoDenomValidation(otherDenom).Add(tokenOut.Amount),
						},
					}

					_, remCoins, err := pool.numSharesOutFromTokensIn(tokensIn)
					require.NoError(t, err)

					// Because of rounding errors, we might receive remcoins up to ~ly/lx
					_, assetX, _ := pool.getPoolAssetAndIndex(swapCoin.Denom)
					_, assetY, _ := pool.getPoolAssetAndIndex(otherDenom)

					maxError := assetX.Token.Amount.ToDec().Quo(assetY.Token.Amount.ToDec())

					require.LessOrEqual(t, remCoins.AmountOf(swapCoin.Denom).Int64(), maxError.TruncateInt64())
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestTokensOutFromExactSharesHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name              string
		pool              Pool
		numSharesIn       sdk.Int
		expectedTokensOut sdk.Coins
	}{
		{
			name: "all coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			numSharesIn: sdk.NewInt(50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 200),
			),
		},
		{
			name: "partial coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			numSharesIn: sdk.NewInt(25),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 100),
			),
		},
		{
			name: "fractional coins withdrawn truncates to int, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1000),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			numSharesIn: sdk.NewInt(25),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 2),
				sdk.NewInt64Coin("foo", 5),
			),
		},
		{
			name: "all coins withdrawn, with exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.MustNewDecFromStr("0.5"),
				},
			},
			numSharesIn: sdk.NewInt(50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 100),
			),
		},
		{
			name: "partial coins withdrawn, with exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
				PoolParams: PoolParams{
					ExitFee: sdk.MustNewDecFromStr("0.5"),
				},
			},
			numSharesIn: sdk.NewInt(25),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 25),
				sdk.NewInt64Coin("foo", 50),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tokensOut, err := tc.pool.TokensOutFromPoolSharesIn(tc.numSharesIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedTokensOut, tokensOut)
		})
	}
}

func TestTokensOutFromExactSharesErrors(t *testing.T) {
	for _, tc := range []struct {
		name        string
		pool        Pool
		numSharesIn sdk.Int
	}{
		{
			name: "zero pool shares",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			},
			numSharesIn: sdk.NewInt(0),
		},
		{
			name: "too many pool shares",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("bar", 100),
					},
					{
						Token: sdk.NewInt64Coin("foo", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			},
			numSharesIn: sdk.NewInt(51),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.pool.TokensOutFromPoolSharesIn(tc.numSharesIn)
			require.Error(t, err)
		})
	}
}

func TestUpdateLiquidityHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		pool                  Pool
		numShares             sdk.Int
		newLiquidity          sdk.Coins
		expectedNumShares     sdk.Int
		expectedNewPoolAssets []PoolAsset
	}{
		{
			name: "all coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			numShares: sdk.NewInt(10),
			newLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(110),
			expectedNewPoolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 110),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 220),
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pool.incrementBalances(tc.numShares, tc.newLiquidity)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNumShares, tc.pool.TotalShares.Amount)
			require.Equal(t, tc.expectedNewPoolAssets, tc.pool.PoolAssets)
		})
	}
}

func TestUpdateLiquidityInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name         string
		pool         Pool
		numShares    sdk.Int
		newLiquidity sdk.Coins
	}{
		{
			name: "add non-existent coin",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			numShares: sdk.NewInt(10),
			newLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 20),
			),
		},
		{
			name: "no existing liquidity",
			pool: Pool{
				PoolAssets:  []PoolAsset{},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			numShares: sdk.NewInt(10),
			newLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 20),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pool.incrementBalances(tc.numShares, tc.newLiquidity)
			require.Error(t, err)
		})
	}
}
