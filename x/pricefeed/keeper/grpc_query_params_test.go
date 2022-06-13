package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := testutilkeeper.PricefeedKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.Params{
		Pairs: []types.Pair{
			{Token1: "btc", Token0: "usd", Oracles: nil, Active: true},
			{Token1: "xrp", Token0: "usd", Oracles: nil, Active: true},
		},
	}
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
