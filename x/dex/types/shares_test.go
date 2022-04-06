package types

import (
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
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", tc.existingShares),
			}
			numShares, remCoins, _ := pool.maximalSharesFromExactRatioJoin(tc.tokensIn)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
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
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 100),
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
			err := tc.pool.updateLiquidity(tc.numShares, tc.newLiquidity)
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
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 100),
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
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 100),
			},
			numShares: sdk.NewInt(10),
			newLiquidity: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 20),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pool.updateLiquidity(tc.numShares, tc.newLiquidity)
			require.Error(t, err)
		})
	}
}
