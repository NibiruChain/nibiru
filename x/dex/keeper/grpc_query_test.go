package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	response, err := app.DexKeeper.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestQueryPoolHappyPath(t *testing.T) {
	tests := []struct {
		name         string
		existingPool types.Pool
	}{
		{
			name: "correct fetch pool",
			existingPool: types.Pool{
				Id:      1,
				Address: sample.AccAddress().String(),
				PoolParams: types.PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.03"),
					ExitFee: sdk.MustNewDecFromStr("0.03"),
				},
				PoolAssets: []types.PoolAsset{
					{
						Token:  sdk.NewInt64Coin("bar", 100),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bar", 100),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
				TotalShares: sdk.NewInt64Coin("matrix/pool/1", 200),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewMatrixApp(true)
			app.DexKeeper.SetPool(ctx, tc.existingPool)

			resp, err := app.DexKeeper.Pool(sdk.WrapSDKContext(ctx), &types.QueryPoolRequest{
				PoolId: 1,
			})
			require.NoError(t, err)
			require.Equal(t, tc.existingPool, *resp.Pool)
		})
	}
}

func TestQueryPoolFail(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "invalid request",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewMatrixApp(true)
			resp, err := app.DexKeeper.Pool(sdk.WrapSDKContext(ctx), nil)
			require.Error(t, err)
			require.Nil(t, resp)
		})
	}
}
