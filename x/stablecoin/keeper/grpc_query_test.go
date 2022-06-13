package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"
)

func TestParamsQuery(t *testing.T) {
	nibiruApp, ctx := testutilapp.NewNibiruApp(true)
	keeper := &nibiruApp.StablecoinKeeper
	wctx := sdk.WrapSDKContext(ctx)
	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(
		wctx, &types.QueryParamsRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
