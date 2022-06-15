package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
)

func TestOraclesQuery(t *testing.T) {
	keeper, ctx := testkeeper.PricefeedKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.Params{
		Pairs: []types.Pair{
			{Token1: "btc", Token0: "usd", Oracles: nil, Active: true},
			{Token1: "xrp", Token0: "usd", Oracles: nil, Active: true},
			{Token1: "ada", Token0: "usd", Oracles: []sdk.AccAddress{
				[]byte("some oracle address"), []byte("some other oracle address"),
			}, Active: true},
			{Token1: "eth", Token0: "usd", Oracles: []sdk.AccAddress{[]byte("random oracle address")}, Active: false},
		},
	}
	keeper.SetParams(ctx, params)

	// Use the ADA pair to query for oracles
	adaPair := params.Pairs[2]
	response, err := keeper.Oracles(wctx, &types.QueryOraclesRequest{PairId: adaPair.PairID()})
	require.NoError(t, err)
	require.Equal(t, &types.QueryOraclesResponse{Oracles: oraclesToAddress(adaPair.Oracles)}, response)

	// Use the ETH pair to query for oracles
	ethPair := params.Pairs[3]
	response, err = keeper.Oracles(wctx, &types.QueryOraclesRequest{PairId: ethPair.PairID()})
	require.NoError(t, err)
	require.Equal(t, &types.QueryOraclesResponse{Oracles: oraclesToAddress(ethPair.Oracles)}, response)
}

func oraclesToAddress(accAddress []sdk.AccAddress) []string {
	r := []string{}
	for _, a := range accAddress {
		r = append(r, a.String())
	}
	return r
}
