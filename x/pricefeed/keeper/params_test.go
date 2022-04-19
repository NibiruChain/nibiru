package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/testkeeper"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.PricefeedKeeper(t)
	params_set := types.Params{
		Pairs: []types.Pair{
			{PairID: "btc:usd", BaseAsset: "btc", QuoteAsset: "usd", Oracles: nil, Active: true},
			{PairID: "xrp:usd", BaseAsset: "xrp", QuoteAsset: "usd", Oracles: nil, Active: true},
			{PairID: "xrp:usd:30", BaseAsset: "xrp", QuoteAsset: "usd", Oracles: nil, Active: true},
		},
	}

	k.SetParams(ctx, params_set)

	require.EqualValues(t, params_set, k.GetParams(ctx))
}
