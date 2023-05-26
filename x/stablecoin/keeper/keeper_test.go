package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
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
				adjustmentStep := sdk.MustNewDecFromStr("0.0035")
				priceLowerBound := sdk.MustNewDecFromStr("0.9990")
				priceUpperBound := sdk.MustNewDecFromStr("1.0002")

				params := types.NewParams(
					collRatio, feeRatio, feeRatioEF, bonusRateRecoll, "15 min", adjustmentStep,
					priceLowerBound,
					priceUpperBound, true)

				return params
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
			stableKeeper := nibiruApp.StablecoinKeeper

			params := tc.requiredParams()
			stableKeeper.SetParams(ctx, params)

			require.EqualValues(t, params, stableKeeper.GetParams(ctx))
		})
	}
}
