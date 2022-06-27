package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := testutilkeeper.PricefeedKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.Params{Pairs: common.NewAssetPairs("btc:usd", "xrp:usd")}
	keeper.SetParams(ctx, params)

	response, err := keeper.QueryParams(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestOraclesQuery(t *testing.T) {
	keeper, ctx := testutilkeeper.PricefeedKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	pairs := common.NewAssetPairs("usd:btc", "usd:xrp", "usd:ada", "usd:eth")
	params := types.Params{Pairs: pairs}
	keeper.SetParams(ctx, params)

	_, addrs := sample.PrivKeyAddressPairs(3)
	oracleA, oracleB, oracleC := addrs[0], addrs[1], addrs[2]

	t.Log("whitelist oracles A, B on pair 2")
	keeper.WhitelistOraclesForPairs(
		ctx,
		/*oracles=*/ []sdk.AccAddress{oracleA, oracleB},
		/*pairs=*/ []common.AssetPair{pairs[2]})

	t.Log("whitelist oracle  C    on pair 3")
	keeper.WhitelistOraclesForPairs(
		ctx,
		/*oracles=*/ []sdk.AccAddress{oracleC},
		/*pairs=*/ []common.AssetPair{pairs[3]})

	t.Log("Query for pair 2 oracles | ADA")
	response, err := keeper.QueryOracles(wctx, &types.QueryOraclesRequest{
		PairId: pairs[2].String()})
	require.NoError(t, err)
	require.Equal(t, &types.QueryOraclesResponse{
		Oracles: []string{oracleA.String(), oracleB.String()}}, response)

	t.Log("Query for pair 3 oracles | ETH")
	response, err = keeper.QueryOracles(wctx, &types.QueryOraclesRequest{
		PairId: pairs[3].String()})
	require.NoError(t, err)
	require.Equal(t, &types.QueryOraclesResponse{
		Oracles: []string{oracleC.String()}}, response)
}

func TestMarketsQuery(t *testing.T) {
	keeper, ctx := testutilkeeper.PricefeedKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	pairs := common.NewAssetPairs("btc:usd", "xrp:usd", "ada:usd", "eth:usd")
	params := types.Params{Pairs: pairs}
	keeper.SetParams(ctx, params)

	t.Log("Give pairs 2 and 3 distinct oracles")
	oracle2, oracle3 := sample.AccAddress(), sample.AccAddress()
	keeper.OraclesStore().AddOracles(ctx, pairs[2], []sdk.AccAddress{oracle2})
	keeper.OraclesStore().AddOracles(ctx, pairs[3], []sdk.AccAddress{oracle3})

	t.Log("Set all pairs but 3 active")
	keeper.ActivePairsStore().SetMany(ctx, pairs[:3], true)
	keeper.ActivePairsStore().SetMany(ctx, common.AssetPairs{pairs[3]}, false)

	queryResp, err := keeper.QueryMarkets(wctx, &types.QueryMarketsRequest{})
	require.NoError(t, err)
	wantQueryResponse := &types.QueryMarketsResponse{
		Markets: []types.Market{
			{
				PairID:  pairs[0].String(),
				Oracles: []string(nil),
				Active:  true,
			},
			{
				PairID:  pairs[1].String(),
				Oracles: []string(nil),
				Active:  true,
			},
			{
				PairID:  pairs[2].String(),
				Oracles: []string{oracle2.String()},
				Active:  true,
			},
			{
				PairID:  pairs[3].String(),
				Oracles: []string{oracle3.String()},
				Active:  false,
			},
		},
	}
	for idx, wantMarket := range wantQueryResponse.Markets {
		assert.EqualValues(t, wantMarket, queryResp.Markets[idx])
	}
}
