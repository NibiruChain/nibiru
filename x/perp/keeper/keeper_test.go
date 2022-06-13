package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/perp/types"
	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"

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
				params := types.Params{
					Stopped:                true,
					MaintenanceMarginRatio: sdk.OneDec(),
				}
				return params
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutilapp.NewNibiruApp(true)
			perpKeeper := &nibiruApp.PerpKeeper

			params := tc.requiredParams()
			perpKeeper.SetParams(ctx, params)

			require.EqualValues(t, params, perpKeeper.GetParams(ctx))
		})
	}
}

func TestGetAndSetParams_Errors(t *testing.T) {
	t.Run("Calling Get without setting causes a panic", func(t *testing.T) {
		nibiruApp, ctx := testutilapp.NewNibiruApp(false)
		perpKeeper := &nibiruApp.PerpKeeper

		require.Panics(
			t,
			func() { perpKeeper.GetParams(ctx) },
		)
	})
}
