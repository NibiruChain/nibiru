package common_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestNewAssetPair_Constructor(t *testing.T) {
	tests := []struct {
		name      string
		tokenPair string
		err       error
	}{
		{
			"only one token",
			common.DenomNIBI,
			common.ErrInvalidTokenPair,
		},
		{
			"more than 2 tokens",
			fmt.Sprintf("%s%s%s%s%s", common.DenomNIBI, common.PairSeparator, common.DenomNUSD,
				common.PairSeparator, common.DenomUSDC),
			common.ErrInvalidTokenPair,
		},
		{
			"different separator",
			fmt.Sprintf("%s%s%s", common.DenomNIBI, "%", common.DenomNUSD),
			common.ErrInvalidTokenPair,
		},
		{
			"correct pair",
			fmt.Sprintf("%s%s%s", common.DenomNIBI, common.PairSeparator, common.DenomNUSD),
			nil,
		},
		{
			"empty token identifier",
			fmt.Sprintf("%s%s%s", "", common.PairSeparator, "eth"),
			fmt.Errorf("empty token identifiers are not allowed"),
		},
		{
			"invalid denom 1",
			fmt.Sprintf("-invalid1%svalid", common.PairSeparator),
			fmt.Errorf("invalid denom"),
		},
		{
			"invalid denom 2",
			fmt.Sprintf("valid%s-invalid2", common.PairSeparator),
			fmt.Errorf("invalid denom"),
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

	require.Equal(t, "uatom", pair.BaseDenom())
	require.Equal(t, "unibi", pair.QuoteDenom())
}

func TestAssetPair_Marshaling(t *testing.T) {
	testCases := []testutil.FunctionTestCase{
		{
			Name: "verbose equal suite",
			Test: func() {
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
			Name: "panics suite",
			Test: func() {
				require.Panics(t, func() {
					common.MustNewAssetPair("aaa:bbb:ccc")
				})
			},
		},
	}

	testutil.RunFunctionTests(t, testCases)
}
