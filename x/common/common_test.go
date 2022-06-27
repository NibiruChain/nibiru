package common_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
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
			outPoolName := common.SortedPairNameFromDenoms(tc.denoms)
			require.Equal(t, tc.poolName, outPoolName)
		})
	}
}

func TestAssetPair_InverseAndSort(t *testing.T) {
	testCases := []struct {
		name   string
		pair   common.AssetPair
		proper bool
	}{
		{
			name:   "proper and improper order pairs are inverses-1",
			pair:   common.AssetPair{Token0: "atom", Token1: "osmo"},
			proper: true,
		},
		{
			name:   "proper and improper order pairs are inverses-2",
			pair:   common.AssetPair{Token0: "osmo", Token1: "atom"},
			proper: false,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			if tc.proper {
				require.Truef(t, tc.pair.IsSortedOrder(),
					"pair: '%v' name: '%v'", tc.pair.String(), tc.pair.SortedName())
				require.EqualValues(t, tc.pair.SortedName(), tc.pair.String())
			} else {
				require.Truef(t, tc.pair.Inverse().IsSortedOrder(),
					"pair: '%v' name: '%v'", tc.pair.String(), tc.pair.SortedName())
				require.EqualValues(t, tc.pair.SortedName(), tc.pair.Inverse().String())
			}

			require.True(t, true)
		})
	}
}

func TestNewAssetPair_Constructor(t *testing.T) {
	tests := []struct {
		name      string
		tokenPair string
		err       error
	}{
		{
			"only one token",
			common.DenomGov,
			common.ErrInvalidTokenPair,
		},
		{
			"more than 2 tokens",
			fmt.Sprintf("%s%s%s%s%s", common.DenomGov, common.PairSeparator, common.DenomStable,
				common.PairSeparator, common.DenomColl),
			common.ErrInvalidTokenPair,
		},
		{
			"different separator",
			fmt.Sprintf("%s%s%s", common.DenomGov, "%", common.DenomStable),
			common.ErrInvalidTokenPair,
		},
		{
			"correct pair",
			fmt.Sprintf("%s%s%s", common.DenomGov, common.PairSeparator, common.DenomStable),
			nil,
		},
		{
			"empty token identifier",
			fmt.Sprintf("%s%s%s", "", common.PairSeparator, "eth"),
			fmt.Errorf("empty token identifiers are not allowed"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := common.NewAssetPair(tc.tokenPair)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAsset_GetQuoteBaseToken(t *testing.T) {
	pair, err := common.NewAssetPair("uatom:unibi")
	require.NoError(t, err)

	require.Equal(t, "uatom", pair.GetBaseTokenDenom())
	require.Equal(t, "unibi", pair.GetQuoteTokenDenom())
}

func TestAssetPair_Marshaling(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "verbose equal suite",
			test: func() {
				pair := common.MustNewAssetPair("abc:xyz")
				matchingOther := common.MustNewAssetPair("abc:xyz")
				mismatchToken1 := common.MustNewAssetPair("abc:abc")
				inversePair := common.MustNewAssetPair("xyz:abc")

				require.NoError(t, (&pair).VerboseEqual(&matchingOther))
				require.True(t, (&pair).Equal(&matchingOther))

				require.Error(t, (&pair).VerboseEqual(&inversePair))
				require.False(t, (&pair).Equal(&inversePair))

				require.Error(t, (&pair).VerboseEqual(&mismatchToken1))
				require.True(t, !(&pair).Equal(&mismatchToken1))

				require.Error(t, (&pair).VerboseEqual(pair.String()))
				require.False(t, (&pair).Equal(&mismatchToken1))
			},
		},
		{
			name: "panics suite",
			test: func() {
				require.Panics(t, func() {
					common.MustNewAssetPair("aaa:bbb:ccc")
				})
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestAssetPairs_Contains(t *testing.T) {
	pairs := common.AssetPairs{
		common.PairBTCStable, common.PairETHStable,
	}

	pair := common.PairGovStable
	isContained, atIdx := pairs.ContainsAtIndex(pair)
	assert.False(t, isContained)
	assert.Equal(t, -1, atIdx)
	assert.False(t, pairs.Contains(pair))

	pair = pairs[0]
	isContained, atIdx = pairs.ContainsAtIndex(pair)
	assert.True(t, isContained)
	assert.Equal(t, 0, atIdx)
	assert.True(t, pairs.Contains(pair))
}
