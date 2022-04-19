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
			{Token1: "btc", Token0: "usd", Oracles: nil, Active: true},
			{Token1: "xrp", Token0: "usd", Oracles: nil, Active: true},
		},
	}

	k.SetParams(ctx, params_set)

	require.EqualValues(t, params_set, k.GetParams(ctx))
}
