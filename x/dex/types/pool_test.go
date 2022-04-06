package types

import (
	"testing"

	"github.com/MatrixDao/matrix/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGetPoolShareBaseDenom(t *testing.T) {
	require.Equal(t, "matrix/pool/123", GetPoolShareBaseDenom(123))
}

func TestGetPoolShareDisplayDenom(t *testing.T) {
	require.Equal(t, "MATRIX-POOL-123", GetPoolShareDisplayDenom(123))
}

func TestNewPool(t *testing.T) {
	poolAccountAddr := sample.AccAddress()
	poolParams := PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []PoolAsset{
		{
			Token:  sdk.NewInt64Coin("foo", 100),
			Weight: sdk.NewInt(1),
		},
		{
			Token:  sdk.NewInt64Coin("bar", 100),
			Weight: sdk.NewInt(1),
		},
	}

	pool, err := NewPool(1 /*=poold*/, poolAccountAddr, poolParams, poolAssets)
	require.NoError(t, err)
	require.Equal(t, Pool{
		Id:         1,
		Address:    poolAccountAddr.String(),
		PoolParams: poolParams,
		PoolAssets: []PoolAsset{
			{
				Token:  sdk.NewInt64Coin("bar", 100),
				Weight: sdk.NewInt(1 << 30),
			},
			{
				Token:  sdk.NewInt64Coin("foo", 100),
				Weight: sdk.NewInt(1 << 30),
			},
		},
		TotalWeight: sdk.NewInt(2 << 30),
		TotalShares: sdk.NewCoin("matrix/pool/1", sdk.NewIntWithDecimal(100, 18)),
	}, pool)
}

func TestJoinPoolHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name              string
		pool              Pool
		tokensIn          sdk.Coins
		expectedNumShares sdk.Int
		expectedRemCoins  sdk.Coins
		expectedPool      Pool
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
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 100),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(10),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 110),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 220),
					},
				},
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 110),
			},
		},
		{
			name: "partial coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 100),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 10),
			),
			expectedNumShares: sdk.NewInt(5),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 5),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 105),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 210),
					},
				},
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 105),
			},
		},
		{
			name: "difficult numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_498_579),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_403_945),
					},
				},
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 1_000_000),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 4859), // 0.138885 % of pool
				sdk.NewInt64Coin("bbb", 1345), // 0.09580147 % of pool
			),
			expectedNumShares: sdk.NewInt(958),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1507),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_501_931),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_405_290),
					},
				},
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 1_000_958),
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			numShares, remCoins, err := tc.pool.JoinPool(tc.tokensIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
			require.Equal(t, tc.expectedPool, tc.pool)
		})
	}
}

func TestJoinPoolInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pool     Pool
		tokensIn sdk.Coins
	}{
		{
			name: "not enough tokens",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 100),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := tc.pool.JoinPool(tc.tokensIn)
			require.Error(t, err)
		})
	}
}
