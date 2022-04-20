package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
)

// Params
func TestGetAndSetParams(t *testing.T) {
	tests := []struct {
		name           string
		requiredParams func() types.Params
	}{
		{
			"get default params",
			func() types.Params {
				return types.DefaultParams()
			},
		},
		{
			"Get non-default params",
			func() types.Params {
				collRatio := sdk.MustNewDecFromStr("0.5")
				feeRatio := collRatio
				feeRatioEF := collRatio
				bonusRateRecoll := sdk.MustNewDecFromStr("0.002")
				params := types.NewParams(
					collRatio, feeRatio, feeRatioEF, bonusRateRecoll)

				return params
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutil.NewNibiruApp(true)
			stableKeeper := nibiruApp.StablecoinKeeper

			params := tc.requiredParams()
			stableKeeper.SetParams(ctx, params)

			require.EqualValues(t, params, stableKeeper.GetParams(ctx))
		})
	}
}

func TestGetAndSetParams_Errors(t *testing.T) {
	t.Run("Calling Get without setting causes a panic", func(t *testing.T) {
		nibiruApp, ctx := testutil.NewNibiruApp(false)
		stableKeeper := nibiruApp.StablecoinKeeper

		require.Panics(
			t,
			func() { stableKeeper.GetParams(ctx) },
		)
	})
}
