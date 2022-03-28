package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/pricefeed/types"
	testkeeper "github.com/MatrixDao/matrix/x/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.PricefeedKeeper(t)
	params_set := types.Params{
		Markets: []types.Market{
			{MarketID: "btc:usd", BaseAsset: "btc", QuoteAsset: "usd", Oracles: nil, Active: true},
			{MarketID: "xrp:usd", BaseAsset: "xrp", QuoteAsset: "usd", Oracles: nil, Active: true},
			{MarketID: "xrp:usd:30", BaseAsset: "xrp", QuoteAsset: "usd", Oracles: nil, Active: true},
		},
	}

	k.SetParams(ctx, params_set)

	require.EqualValues(t, params_set, k.GetParams(ctx))
}
