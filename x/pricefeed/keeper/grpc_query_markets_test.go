package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
)

func TestMarketsQuery(t *testing.T) {
	keeper, ctx := testkeeper.PricefeedKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.Params{
		Pairs: []types.Pair{
			{Token0: "btc", Token1: "usd", Oracles: nil, Active: true},
			{Token0: "xrp", Token1: "usd", Oracles: nil, Active: true},
			{Token0: "ada", Token1: "usd", Oracles: []sdk.AccAddress{[]byte("some oracle address")}, Active: true},
			{Token0: "eth", Token1: "usd", Oracles: []sdk.AccAddress{[]byte("random oracle address")}, Active: false},
		},
	}
	keeper.SetParams(ctx, params)

	response, err := keeper.Pairs(wctx, &types.QueryPairsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryPairsResponse{Pairs: pairToPairResponse(params.Pairs)}, response)
}

func pairToPairResponse(pairs []types.Pair) []types.PairResponse {
	r := []types.PairResponse{}
	for _, p := range pairs {
		r = append(r, types.NewPairResponse(p.Token1, p.Token0, p.Oracles, p.Active))
	}
	return r
}
