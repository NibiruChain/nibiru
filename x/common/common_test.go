package common_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
)

func TestTryNewAssetPair(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tokenPair string
		err       error
	}{
		{
			"only one token",
			denoms.NIBI,
			common.ErrInvalidTokenPair,
		},
		{
			"more than 2 tokens",
			fmt.Sprintf("%s%s%s%s%s", denoms.NIBI, common.PairSeparator, denoms.NUSD,
				common.PairSeparator, denoms.USDC),
			common.ErrInvalidTokenPair,
		},
		{
			"different separator",
			fmt.Sprintf("%s%s%s", denoms.NIBI, "%", denoms.NUSD),
			common.ErrInvalidTokenPair,
		},
		{
			"correct pair",
			fmt.Sprintf("%s%s%s", denoms.NIBI, common.PairSeparator, denoms.NUSD),
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
			_, err := common.TryNewAssetPair(tc.tokenPair)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAssetGetQuoteBaseToken(t *testing.T) {
	pair := common.MustNewAssetPair("uatom:unibi")

	require.Equal(t, "uatom", pair.BaseDenom())
	require.Equal(t, "unibi", pair.QuoteDenom())
}

func TestAssetPairEquals(t *testing.T) {
	pair := common.MustNewAssetPair("abc:xyz")
	matchingOther := common.MustNewAssetPair("abc:xyz")
	mismatchToken1 := common.MustNewAssetPair("abc:abc")
	inversePair := common.MustNewAssetPair("xyz:abc")

	require.True(t, pair.Equal(matchingOther))
	require.False(t, pair.Equal(inversePair))
	require.False(t, pair.Equal(mismatchToken1))
}

func TestMustNewAssetPair(t *testing.T) {
	require.Panics(t, func() {
		common.MustNewAssetPair("aaa:bbb:ccc")
	})

	require.NotPanics(t, func() {
		common.MustNewAssetPair("aaa:bbb")
	})
}

func TestInverse(t *testing.T) {
	pair := common.MustNewAssetPair("abc:xyz")
	inverse := pair.Inverse()
	require.Equal(t, "xyz", inverse.BaseDenom())
	require.Equal(t, "abc", inverse.QuoteDenom())
}
