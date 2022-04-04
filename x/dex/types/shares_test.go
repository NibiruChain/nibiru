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
			name: "some coins deposited",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 200),
				},
			},
			existingShares: 100,
			// limiting coin is 'bbb' which accounts for 5% of pool 'bbb'
			// so all 'bbb' tokens should be taken
			// and exactly 5 'aaa' tokens should be taken
			// but osmosis' math takes only 1 'aaa' token
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
			name: "limited by smallest amount",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 1000),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 1000),
				},
			},
			existingShares: 1000,
			// limiting coin is 'aaa' which accounts for 1% of pool 'aaa'
			// so all 'aaa' tokens should be taken
			// and exactly 10 'bbb' tokens should be taken
			// but osmosis' math takes only 1 'bbb' token
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 50),
			),
			expectedNumShares: sdk.NewInt(10),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("bbb", 40),
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
