package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestGetParams(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "calling GetParams without setting returns default",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				k := nibiruApp.PricefeedKeeper
				require.EqualValues(t, types.DefaultParams(), k.GetParams(ctx))
			},
		},
		{
			name: "params match after manual set and include default",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				k := nibiruApp.PricefeedKeeper
				params := types.Params{
					Pairs: common.NewAssetPairs("btc:usd", "xrp:usd"),
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

func TestWhitelistOracles(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "genesis - no oracle provided",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				pk := &nibiruApp.PricefeedKeeper

				oracle := sample.AccAddress()
				paramsPairs := pk.GetParams(ctx).Pairs
				for _, pair := range paramsPairs {
					require.False(t, pk.IsWhitelistedOracle(ctx, pair.String(), oracle))
				}
				gotOraclesMap := pk.GetOraclesForPairs(ctx, paramsPairs)
				gotOracles := gotOraclesMap[paramsPairs[0]]
				require.EqualValues(t, 3, len(gotOracles))
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1hk04vteklhmtwe0zpt7023p5zcgu49e5v3atyp"))
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi10hj3gq54uxd9l5d6a7sn4dcvhd0l3wdgt2zvyp"))
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1r8gjajmlp9tkff0759rmujv568pa7q6v7u4m3z"))
			},
		},
		{
			name: "multiple oracles whitelisted at different times ",
			test: func() {
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				pk := &nibiruApp.PricefeedKeeper

				paramsPairs := pk.GetParams(ctx).Pairs
				for _, pair := range paramsPairs {
					gotOracles := pk.GetOraclesForPair(ctx, pair.String())
					require.EqualValues(t, 3, len(gotOracles))
					require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1hk04vteklhmtwe0zpt7023p5zcgu49e5v3atyp"))
					require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi10hj3gq54uxd9l5d6a7sn4dcvhd0l3wdgt2zvyp"))
					require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1r8gjajmlp9tkff0759rmujv568pa7q6v7u4m3z"))
				}

				oracleA := sample.AccAddress()
				oracleB := sample.AccAddress()

				wantOracles := []sdk.AccAddress{oracleA}
				pk.WhitelistOracles(ctx, wantOracles)
				gotOraclesMap := pk.GetOraclesForPairs(ctx, paramsPairs)
				gotOracles := gotOraclesMap[paramsPairs[0]]
				require.EqualValues(t, 4, len(gotOracles))
				require.Contains(t, gotOracles, oracleA)
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1hk04vteklhmtwe0zpt7023p5zcgu49e5v3atyp"))
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi10hj3gq54uxd9l5d6a7sn4dcvhd0l3wdgt2zvyp"))
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1r8gjajmlp9tkff0759rmujv568pa7q6v7u4m3z"))
				require.NotContains(t, gotOracles, oracleB)

				wantOracles = []sdk.AccAddress{oracleA, oracleB}
				pk.WhitelistOracles(ctx, wantOracles)
				gotOraclesMap = pk.GetOraclesForPairs(ctx, paramsPairs)
				gotOracles = gotOraclesMap[paramsPairs[0]]
				require.EqualValues(t, 5, len(gotOracles))
				require.Contains(t, gotOracles, oracleA)
				require.Contains(t, gotOracles, oracleB)
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1hk04vteklhmtwe0zpt7023p5zcgu49e5v3atyp"))
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi10hj3gq54uxd9l5d6a7sn4dcvhd0l3wdgt2zvyp"))
				require.Contains(t, gotOracles, sdk.MustAccAddressFromBech32("nibi1r8gjajmlp9tkff0759rmujv568pa7q6v7u4m3z"))
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		},
		)
	}
}

func TestWhitelistOraclesForPairs(t *testing.T) {
	testCases := []struct {
		name          string
		startParams   types.Params
		pairsToSet    common.AssetPairs
		endAssetPairs common.AssetPairs
	}{
		{
			name: "whitelist for specific pairs - happy",
			startParams: types.Params{
				Pairs: common.NewAssetPairs("aaa:usd", "bbb:usd", "oraclepair:usd"),
			},
			pairsToSet: common.NewAssetPairs("oraclepair:usd"),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
			pricefeedKeeper := &nibiruApp.PricefeedKeeper
			pricefeedKeeper.SetParams(ctx, tc.startParams)

			oracles := []sdk.AccAddress{sample.AccAddress(), sample.AccAddress()}
			pricefeedKeeper.WhitelistOraclesForPairs(
				ctx,
				oracles,
				/* pairs */ tc.pairsToSet,
			)

			t.Log("Verify that all 'pairsToSet' have the oracle set.")
			for _, pair := range tc.pairsToSet {
				assert.EqualValues(t,
					oracles,
					pricefeedKeeper.GetOraclesForPair(ctx, pair.String()))
			}

			t.Log("Verify that all pairs outside 'pairsToSet' are unaffected.")
			for _, pair := range tc.startParams.Pairs {
				if !tc.pairsToSet.Contains(pair) {
					assert.EqualValues(t,
						[]sdk.AccAddress{},
						pricefeedKeeper.GetOraclesForPair(ctx, pair.String()))
				}
			}
		})
	}
}
