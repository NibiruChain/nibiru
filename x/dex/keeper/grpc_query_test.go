package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/dex/keeper"
	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	app, ctx := testutil.NewNibiruApp(true)

	params := types.DefaultParams()
	app.DexKeeper.SetParams(ctx, params)

	queryServer := keeper.NewQuerier(app.DexKeeper)

	response, err := queryServer.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
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
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 200),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)
			app.DexKeeper.SetPool(ctx, tc.existingPool)

			queryServer := keeper.NewQuerier(app.DexKeeper)

			resp, err := queryServer.Pool(sdk.WrapSDKContext(ctx), &types.QueryPoolRequest{
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
			app, ctx := testutil.NewNibiruApp(true)
			queryServer := keeper.NewQuerier(app.DexKeeper)
			resp, err := queryServer.Pool(sdk.WrapSDKContext(ctx), nil)
			require.Error(t, err)
			require.Nil(t, resp)
		})
	}
}

func TestQueryPools(t *testing.T) {
	tests := []struct {
		name          string
		existingPools []types.Pool
		pagination    query.PageRequest
		expectedPools []types.Pool
	}{
		{
			name: "successful query single pool",
			existingPools: []types.Pool{
				mock.DexPool(
					/*poolId=*/ 1,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("foo", 100),
						sdk.NewInt64Coin("bar", 100),
					),
					/*shares=*/ 100,
				),
			},
			expectedPools: []types.Pool{
				mock.DexPool(
					/*poolId=*/ 1,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("foo", 100),
						sdk.NewInt64Coin("bar", 100),
					),
					/*shares=*/ 100,
				),
			},
		},
		{
			name: "successful query multiple pools",
			existingPools: []types.Pool{
				mock.DexPool(
					/*poolId=*/ 1,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("foo", 100),
						sdk.NewInt64Coin("bar", 100),
					),
					/*shares=*/ 100,
				),
				mock.DexPool(
					/*poolId=*/ 2,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("bar", 100),
						sdk.NewInt64Coin("baz", 100),
					),
					/*shares=*/ 100,
				),
			},
			expectedPools: []types.Pool{
				mock.DexPool(
					/*poolId=*/ 1,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("foo", 100),
						sdk.NewInt64Coin("bar", 100),
					),
					/*shares=*/ 100,
				),
				mock.DexPool(
					/*poolId=*/ 2,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("bar", 100),
						sdk.NewInt64Coin("baz", 100),
					),
					/*shares=*/ 100,
				),
			},
		},
		{
			name: "query pools with pagination",
			existingPools: []types.Pool{
				mock.DexPool(
					/*poolId=*/ 1,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("foo", 100),
						sdk.NewInt64Coin("bar", 100),
					),
					/*shares=*/ 100,
				),
				mock.DexPool(
					/*poolId=*/ 2,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("bar", 100),
						sdk.NewInt64Coin("baz", 100),
					),
					/*shares=*/ 100,
				),
			},
			pagination: query.PageRequest{
				Limit: 1,
			},
			expectedPools: []types.Pool{
				mock.DexPool(
					/*poolId=*/ 1,
					/*assets=*/ sdk.NewCoins(
						sdk.NewInt64Coin("foo", 100),
						sdk.NewInt64Coin("bar", 100),
					),
					/*shares=*/ 100,
				),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)
			for _, existingPool := range tc.existingPools {
				app.DexKeeper.SetPool(ctx, existingPool)
			}

			queryServer := keeper.NewQuerier(app.DexKeeper)

			resp, err := queryServer.Pools(
				sdk.WrapSDKContext(ctx),
				&types.QueryPoolsRequest{
					Pagination: &tc.pagination,
				},
			)
			require.NoError(t, err)

			var responsePools []types.Pool
			for _, p := range resp.Pools {
				responsePools = append(responsePools, *p)
			}
			require.Equal(t, tc.expectedPools, responsePools)
		})
	}
}

func TestQueryPoolParams(t *testing.T) {
	tests := []struct {
		name               string
		existingPool       types.Pool
		expectedPoolParams types.PoolParams
	}{
		{
			name: "successful fetch pool params",
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
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 200),
			},
			expectedPoolParams: types.PoolParams{
				SwapFee: sdk.MustNewDecFromStr("0.03"),
				ExitFee: sdk.MustNewDecFromStr("0.03"),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)
			app.DexKeeper.SetPool(ctx, tc.existingPool)

			queryServer := keeper.NewQuerier(app.DexKeeper)

			resp, err := queryServer.PoolParams(sdk.WrapSDKContext(ctx), &types.QueryPoolParamsRequest{
				PoolId: 1,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expectedPoolParams, *resp.PoolParams)
		})
	}
}
