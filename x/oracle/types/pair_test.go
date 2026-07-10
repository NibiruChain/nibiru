package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/nutil/denoms"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

func TestTryNewPair(t *testing.T) {
	tests := []struct {
		name     string
		tokenStr string
		err      error
	}{
		{
			"only one token",
			appconst.DENOM_UNIBI,
			types.ErrInvalidTokenPair,
		},
		{
			"more than 2 tokens",
			fmt.Sprintf("%s:%s:%s", appconst.DENOM_UNIBI, denoms.NUSD, denoms.USDC),
			types.ErrInvalidTokenPair,
		},
		{
			"different separator",
			fmt.Sprintf("%s,%s", appconst.DENOM_UNIBI, denoms.NUSD),
			types.ErrInvalidTokenPair,
		},
		{
			"correct pair",
			fmt.Sprintf("%s:%s", appconst.DENOM_UNIBI, denoms.NUSD),
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
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.TryNewPair(tc.tokenStr)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetDenoms(t *testing.T) {
	pair := types.MustNewPair("uatom:unibi")

	require.Equal(t, "uatom", pair.BaseDenom())
	require.Equal(t, "unibi", pair.QuoteDenom())
}

func TestEquals(t *testing.T) {
	pair := types.MustNewPair("abc:xyz")
	matchingOther := types.MustNewPair("abc:xyz")
	mismatchToken1 := types.MustNewPair("abc:abc")
	inversePair := types.MustNewPair("xyz:abc")

	require.True(t, pair.Equal(matchingOther))
	require.False(t, pair.Equal(inversePair))
	require.False(t, pair.Equal(mismatchToken1))
}

func TestMustNewAssetPair(t *testing.T) {
	require.Panics(t, func() {
		types.MustNewPair("aaa:bbb:ccc")
	})

	require.NotPanics(t, func() {
		types.MustNewPair("aaa:bbb")
	})
}

func TestInverse(t *testing.T) {
	pair := types.MustNewPair("abc:xyz")
	inverse := pair.Inverse()
	require.Equal(t, "xyz", inverse.BaseDenom())
	require.Equal(t, "abc", inverse.QuoteDenom())
}

func TestMarshalJSON(t *testing.T) {
	cdc := app.MakeEncodingConfig()

	testCases := []struct {
		name      string
		input     types.Pair
		strOutput string
	}{
		{name: "happy-0", input: types.Pair("abc:xyz"), strOutput: "\"abc:xyz\""},
		{name: "happy-1", input: types.Pair("abc:xyz:foo"), strOutput: "\"abc:xyz:foo\""},
		{name: "happy-2", input: types.Pair("abc"), strOutput: "\"abc\""},
		{name: "empty", input: types.Pair(""), strOutput: "\"\""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBz, err := cdc.Amino.MarshalJSON(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.strOutput, string(jsonBz))

			jsonBzCustom, err := tc.input.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, jsonBzCustom, jsonBz)

			newPair := new(types.Pair)
			require.NoError(t, cdc.Amino.UnmarshalJSON(jsonBz, newPair))
			require.Equal(t, tc.input, *newPair)

			newNewPair := new(types.Pair)
			*newNewPair = tc.input
			require.NoError(t, newNewPair.UnmarshalJSON(jsonBz))

			bz, err := tc.input.Marshal()
			require.NoError(t, err)
			newNewNewPair := new(types.Pair)
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
				pairs := types.MustNewPairs(tc.pairStrs...)
				newPairStrs := types.PairsToStrings(pairs)
				require.Equal(t, tc.pairStrs, newPairStrs)
			})
		})
	}
}
