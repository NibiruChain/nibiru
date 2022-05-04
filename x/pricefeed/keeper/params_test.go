package keeper_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "calling GetParams without setting returns default",
			test: func() {
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				k := nibiruApp.PriceKeeper
				require.EqualValues(t, types.DefaultParams(), k.GetParams(ctx))
			},
		},
		{
			name: "params match after manual set and include default",
			test: func() {
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				k := nibiruApp.PriceKeeper
				params := types.Params{
					Pairs: []types.Pair{
						{Token1: "btc", Token0: "usd", Oracles: nil, Active: true},
						{Token1: "xrp", Token0: "usd", Oracles: nil, Active: true},
					},
				}
				k.SetParams(ctx, params)
				require.EqualValues(t, params, k.GetParams(ctx))

				params.Pairs = append(params.Pairs, types.DefaultPairs...)
				k.SetParams(ctx, params)
				require.EqualValues(t, params, k.GetParams(ctx))
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}


	require.EqualValues(t, params_set, k.GetParams(ctx))
}
