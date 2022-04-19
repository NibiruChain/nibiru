package common_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/stretchr/testify/require"
)

func TestPairNameFromDenoms(t *testing.T) {
	testCases := []struct {
		name     string
		denoms   []string
		poolName string
	}{
		{
			name:     "ATOM:OSMO in correct order",
			denoms:   []string{"atom", "osmo"},
			poolName: "atom:osmo",
		},
		{
			name:     "ATOM:OSMO in wrong order",
			denoms:   []string{"osmo", "atom"},
			poolName: "atom:osmo",
		},
		{
			name:     "X:Y:Z in correct order",
			denoms:   []string{"x", "y", "z"},
			poolName: "x:y:z",
		},
		{
			name:     "X:Y:Z in wrong order",
			denoms:   []string{"z", "x", "y"},
			poolName: "x:y:z",
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			outPoolName := common.PoolNameFromDenoms(tc.denoms)
			require.Equal(t, tc.poolName, outPoolName)
		})
	}
}

func TestAssetPair(t *testing.T) {
	testCases := []struct {
		name   string
		pair   common.AssetPair
		proper bool
	}{
		{
			name:   "proper and improper order pairs are inverses-1",
			pair:   common.AssetPair{"atom", "osmo"},
			proper: true,
		},
		{
			name:   "proper and improper order pairs are inverses-2",
			pair:   common.AssetPair{"osmo", "atom"},
			proper: false,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			if tc.proper {
				require.True(t, tc.pair.IsProperOrder())
				require.Equal(t, tc.pair.Name(), tc.pair.String())
			} else {
				require.True(t, tc.pair.Inverse().IsProperOrder())
				require.Equal(t, tc.pair.Name(), tc.pair.Inverse().String())
			}

			require.True(t, true)
		})
	}
}
