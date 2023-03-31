package types

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCalSpotPrice(t *testing.T) {
	tests := []struct {
		name          string
		poolAssets    []PoolAsset
		expectedPrice sdk.Dec
	}{
		{
			"equal weight: 2 tokens",
			[]PoolAsset{
				{
					Token:  sdk.NewInt64Coin("foo", 2*common.TO_MICRO),
					Weight: sdk.NewInt(100),
				},
				{
					Token:  sdk.NewInt64Coin("bar", 1*common.TO_MICRO),
					Weight: sdk.NewInt(100),
				},
			},
			sdk.MustNewDecFromStr("2"),
		},
		{
			"different weight: 2 tokens",
			[]PoolAsset{
				{
					Token:  sdk.NewInt64Coin("foo", 2*common.TO_MICRO),
					Weight: sdk.NewInt(80),
				},
				{
					Token:  sdk.NewInt64Coin("bar", 1*common.TO_MICRO),
					Weight: sdk.NewInt(20),
				},
			},
			sdk.MustNewDecFromStr("0.5"),
		},
		{
			"equal weight: 3 tokens",
			[]PoolAsset{
				{
					Token:  sdk.NewInt64Coin("foo", 2*common.TO_MICRO),
					Weight: sdk.NewInt(100),
				},
				{
					Token:  sdk.NewInt64Coin("goo", 1*common.TO_MICRO),
					Weight: sdk.NewInt(100),
				},
				{
					Token:  sdk.NewInt64Coin("bar", 1*common.TO_MICRO),
					Weight: sdk.NewInt(100),
				},
			},
			sdk.MustNewDecFromStr("2"),
		},
		{
			"different weight: 3 tokens",
			[]PoolAsset{
				{
					Token:  sdk.NewInt64Coin("foo", 2*common.TO_MICRO),
					Weight: sdk.NewInt(60),
				},
				{
					Token:  sdk.NewInt64Coin("bar", 1*common.TO_MICRO),
					Weight: sdk.NewInt(20),
				},
				{
					Token:  sdk.NewInt64Coin("foobar", 1*common.TO_MICRO),
					Weight: sdk.NewInt(20),
				},
			},
			sdk.MustNewDecFromStr("0.666666666666666667"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			poolAccountAddr := testutil.AccAddress()
			poolParams := PoolParams{
				SwapFee:  sdk.NewDecWithPrec(3, 2),
				ExitFee:  sdk.NewDecWithPrec(3, 2),
				PoolType: PoolType_BALANCER,
			}

			pool, err := NewPool(1, poolAccountAddr, poolParams, tc.poolAssets)
			require.NoError(t, err)

			actualPrice, err := pool.CalcSpotPrice(tc.poolAssets[0].Token.Denom, tc.poolAssets[1].Token.Denom)
			require.NoError(t, err)

			require.Equal(t, tc.expectedPrice, actualPrice)
		})
	}
}
