package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/testutil/testkeeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := testkeeper.PricefeedKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.Params{
		Markets: []types.Market{
			{MarketID: "btc:usd", BaseAsset: "btc", QuoteAsset: "usd", Oracles: nil, Active: true},
			{MarketID: "xrp:usd", BaseAsset: "xrp", QuoteAsset: "usd", Oracles: nil, Active: true},
			{MarketID: "xrp:usd:30", BaseAsset: "xrp", QuoteAsset: "usd", Oracles: nil, Active: true},
		},
	}
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
