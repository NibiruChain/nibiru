package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/keeper"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

func TestQueryFeeTokens(t *testing.T) {
	testCases := []struct {
		name     string
		malleate func(app *app.NibiruApp, ctx sdk.Context)
	}{
		{
			name: "success: query fee tokens",
			malleate: func(app *app.NibiruApp, ctx sdk.Context) {
				err := app.GasTokenKeeper.SetFeeTokens(ctx, validFeeTokens)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			tc.malleate(nibiruApp, ctx)
			queryHelper := baseapp.NewQueryServerTestHelper(
				ctx, nibiruApp.InterfaceRegistry(),
			)
			types.RegisterQueryServer(queryHelper, keeper.NewQuerier(nibiruApp.GasTokenKeeper))
			queryClient := types.NewQueryClient(queryHelper)
			resp, err := queryClient.FeeTokens(sdk.WrapSDKContext(ctx), &types.QueryFeeTokensRequest{})
			require.NoError(t, err)

			expected := append([]types.FeeToken(nil), validFeeTokens...)
			sortFeeTokens(expected)
			sortFeeTokens(resp.FeeTokens)
			require.Equal(t, expected, resp.FeeTokens)
		})
	}

}
