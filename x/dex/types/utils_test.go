package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGetPoolAssetAndIndexHappyPath(t *testing.T) {
	tests := []struct {
		name              string
		poolAssets        []PoolAsset
		denom             string
		expectedPoolAsset PoolAsset
		expectedIndex     int
	}{
		{
			name: "happy path single asset",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
			},
			denom: "foo",
			expectedPoolAsset: PoolAsset{
				Token:  sdk.NewInt64Coin("foo", 100),
				Weight: sdk.NewInt(1),
			},
			expectedIndex: 0,
		},
		{
			name: "happy path multiple asset",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("bar", 100),
					Weight: sdk.NewInt(1),
				},
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
				PoolAsset{
					Token:  sdk.NewInt64Coin("zee", 100),
					Weight: sdk.NewInt(1),
				},
			},
			denom: "foo",
			expectedPoolAsset: PoolAsset{
				Token:  sdk.NewInt64Coin("foo", 100),
				Weight: sdk.NewInt(1),
			},
			expectedIndex: 1,
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			index, poolAsset, err := getPoolAssetAndIndex(tc.poolAssets, tc.denom)
			require.NoError(t, err)
			require.Equal(t, tc.expectedIndex, index)
			require.Equal(t, tc.expectedPoolAsset, poolAsset)
		})
	}
}

func TestGetPoolAssetAndIndexErrors(t *testing.T) {
	tests := []struct {
		name          string
		poolAssets    []PoolAsset
		denom         string
		expectedError string
	}{
		{
			name:          "empty pool assets",
			poolAssets:    []PoolAsset{},
			denom:         "foo",
			expectedError: "Empty pool assets.",
		},
		{
			name: "empty denom",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
			},
			denom:         "",
			expectedError: "Empty denom.",
		},
		{
			name: "denom not found - input denom lexicographically higher",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("bar", 100),
					Weight: sdk.NewInt(1),
				},
			},
			denom:         "foo",
			expectedError: "Did not find the PoolAsset (foo)",
		},
		{
			name: "denom not found - input denom lexicographically lower",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
			},
			denom:         "bar",
			expectedError: "Did not find the PoolAsset (bar)",
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := getPoolAssetAndIndex(tc.poolAssets, tc.denom)
			require.Errorf(t, err, tc.expectedError)
		})
	}
}

func TestPoolAssetsCoins(t *testing.T) {
	tests := []struct {
		name          string
		poolAssets    []PoolAsset
		expectedCoins sdk.Coins
	}{
		{
			name: "happy path single asset",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
			},
			expectedCoins: sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
		},
		{
			name: "happy path multiple asset",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("bar", 100),
					Weight: sdk.NewInt(1),
				},
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 200),
					Weight: sdk.NewInt(1),
				},
			},
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 200),
			),
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			coins := poolAssetsCoins(tc.poolAssets)
			require.Equal(t, tc.expectedCoins, coins)
		})
	}
}

func TestSortPoolAssets(t *testing.T) {
	tests := []struct {
		name              string
		poolAssets        []PoolAsset
		expectedPoolAsset []PoolAsset
	}{
		{
			name: "single asset",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
			},
			expectedPoolAsset: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
			},
		},
		{
			name: "happy path multiple asset",
			poolAssets: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
				PoolAsset{
					Token:  sdk.NewInt64Coin("bar", 200),
					Weight: sdk.NewInt(1),
				},
			},
			expectedPoolAsset: []PoolAsset{
				PoolAsset{
					Token:  sdk.NewInt64Coin("bar", 200),
					Weight: sdk.NewInt(1),
				},
				PoolAsset{
					Token:  sdk.NewInt64Coin("foo", 100),
					Weight: sdk.NewInt(1),
				},
			},
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			sortPoolAssetsByDenom(tc.poolAssets)
			require.Equal(t, tc.expectedPoolAsset, tc.poolAssets)
		})
	}
}
