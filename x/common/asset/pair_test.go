package asset_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/codec"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
)

func TestTryNewPair(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tokenStr string
		err      error
	}{
		{
			"only one token",
			denoms.NIBI,
			asset.ErrInvalidTokenPair,
		},
		{
			"more than 2 tokens",
			fmt.Sprintf("%s:%s:%s", denoms.NIBI, denoms.NUSD, denoms.USDC),
			asset.ErrInvalidTokenPair,
		},
		{
			"different separator",
			fmt.Sprintf("%s,%s", denoms.NIBI, denoms.NUSD),
			asset.ErrInvalidTokenPair,
		},
		{
			"correct pair",
			fmt.Sprintf("%s:%s", denoms.NIBI, denoms.NUSD),
			nil,
		},
		{
			"empty token identifier",
			fmt.Sprintf(":%s", denoms.ETH),
			fmt.Errorf("empty token identifiers are not allowed"),
		},
		{
			"invalid denom 1",
			"-invalid1:valid",
			fmt.Errorf("invalid denom"),
		},
		{
			"invalid denom 2",
			"valid:-invalid2",
			fmt.Errorf("invalid denom"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := asset.TryNewPair(tc.tokenStr)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetDenoms(t *testing.T) {
	pair := asset.MustNewPair("uatom:unibi")

	require.Equal(t, "uatom", pair.BaseDenom())
	require.Equal(t, "unibi", pair.QuoteDenom())
}

func TestEquals(t *testing.T) {
	pair := asset.MustNewPair("abc:xyz")
	matchingOther := asset.MustNewPair("abc:xyz")
	mismatchToken1 := asset.MustNewPair("abc:abc")
	inversePair := asset.MustNewPair("xyz:abc")

	require.True(t, pair.Equal(matchingOther))
	require.False(t, pair.Equal(inversePair))
	require.False(t, pair.Equal(mismatchToken1))
}

func TestMustNewAssetPair(t *testing.T) {
	require.Panics(t, func() {
		asset.MustNewPair("aaa:bbb:ccc")
	})

	require.NotPanics(t, func() {
		asset.MustNewPair("aaa:bbb")
	})
}

func TestInverse(t *testing.T) {
	pair := asset.MustNewPair("abc:xyz")
	inverse := pair.Inverse()
	require.Equal(t, "xyz", inverse.BaseDenom())
	require.Equal(t, "abc", inverse.QuoteDenom())
}

func TestMarshalJSON(t *testing.T) {
	cdc := codec.MakeEncodingConfig()

	testCases := []struct {
		name      string
		input     asset.Pair
		strOutput string
	}{
		{name: "happy-0", input: asset.Pair("abc:xyz"), strOutput: "\"abc:xyz\""},
		{name: "happy-1", input: asset.Pair("abc:xyz:foo"), strOutput: "\"abc:xyz:foo\""},
		{name: "happy-2", input: asset.Pair("abc"), strOutput: "\"abc\""},
		{name: "empty", input: asset.Pair(""), strOutput: "\"\""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// MarshalJSON with codec.LegacyAmino
			jsonBz, err := cdc.Amino.MarshalJSON(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.strOutput, string(jsonBz))

			// MarshalJSON on custom type
			jsonBzCustom, err := tc.input.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, jsonBzCustom, jsonBz)

			// UnmarshalJSON with codec.LegacyAmino
			newPair := new(asset.Pair)
			require.NoError(t, cdc.Amino.UnmarshalJSON(jsonBz, newPair))
			require.Equal(t, tc.input, *newPair)

			// UnmarshalJSON on custom type
			newNewPair := new(asset.Pair)
			*newNewPair = tc.input
			require.NoError(t, newNewPair.UnmarshalJSON(jsonBz))

			// Marshal and Unmarshal (to bytes) test
			bz, err := tc.input.Marshal()
			require.NoError(t, err)
			newNewNewPair := new(asset.Pair)
			require.NoError(t, newNewNewPair.Unmarshal(bz))
			require.Equal(t, tc.input, *newNewNewPair)
		})
	}
}

func TestPairsUtils(t *testing.T) {
	testCases := []struct {
		pairStrs    []string
		expectPanic bool
	}{
		{pairStrs: []string{"eth:usd", "btc:usd", "atom:usd"}, expectPanic: false},
		{pairStrs: []string{"eth:usd", "", "abc"}, expectPanic: true},
		{pairStrs: []string{"eth:usd:ftt", "btc:usd"}, expectPanic: true},
	}

	var panicTestFn func(t require.TestingT, f assert.PanicTestFunc, msgAndArgs ...interface{})

	for idx, tc := range testCases {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			if tc.expectPanic {
				panicTestFn = require.Panics
			} else {
				panicTestFn = require.NotPanics
			}
			panicTestFn(t, func() {
				pairs := asset.MustNewPairs(tc.pairStrs...)
				newPairStrs := asset.PairsToStrings(pairs)
				require.Equal(t, tc.pairStrs, newPairStrs)
			})
		})
	}
}
