package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/perp/types"
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
				params := types.Params{
					Stopped:                 true,
					FeePoolFeeRatio:         sdk.OneDec(),
					EcosystemFundFeeRatio:   sdk.OneDec(),
					LiquidationFeeRatio:     sdk.OneDec(),
					PartialLiquidationRatio: sdk.OneDec(),
					TwapLookbackWindow:      15 * time.Minute,
				}
				return params
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
			perpKeeper := &nibiruApp.PerpKeeper

			params := tc.requiredParams()
			perpKeeper.SetParams(ctx, params)

			require.EqualValues(t, params, perpKeeper.GetParams(ctx))
		})
	}
}

func TestGetAndSetParams_Errors(t *testing.T) {
	t.Run("Calling Get without setting causes a panic", func(t *testing.T) {
		nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(false)
		perpKeeper := &nibiruApp.PerpKeeper

		require.Panics(
			t,
			func() { perpKeeper.GetParams(ctx) },
		)
	})
}
