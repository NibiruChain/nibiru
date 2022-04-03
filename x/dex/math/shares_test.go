package math

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMaximalSharesFromExactRatioJoin(t *testing.T) {
	for _, tc := range []struct {
		name              string
		poolAssets        []types.PoolAsset
		existingShares    int64
		tokensIn          sdk.Coins
		expectedNumShares sdk.Int
		expectedRemCoins  sdk.Coins
	}{
		{
			name: "all coins deposited",
			poolAssets: []types.PoolAsset{
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
			poolAssets: []types.PoolAsset{
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
			poolAssets: []types.PoolAsset{
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
			name: "right number of LP shares",
			poolAssets: []types.PoolAsset{
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
			pool := types.Pool{
				Id:          1,
				Address:     "some_address",
				PoolParams:  types.PoolParams{},
				PoolAssets:  tc.poolAssets,
				TotalWeight: sdk.OneInt(),
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", tc.existingShares),
			}
			numShares, remCoins, _ := maximalSharesFromExactRatioJoin(pool, tc.tokensIn)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
		})
	}
}
