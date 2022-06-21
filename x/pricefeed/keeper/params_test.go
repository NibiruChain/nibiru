package keeper_test

import (
	"io/ioutil"
	"testing"

	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simappparams "github.com/cosmos/ibc-go/v3/testing/simapp/params"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestGetParams(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "calling GetParams without setting returns default",
			test: func() {
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				k := nibiruApp.PricefeedKeeper
				require.EqualValues(t, types.DefaultParams(), k.GetParams(ctx))
			},
		},
		{
			name: "params match after manual set and include default",
			test: func() {
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
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
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				pk := &nibiruApp.PricefeedKeeper

				oracle := sample.AccAddress()
				paramsPairs := pk.GetParams(ctx).Pairs
				for _, pair := range paramsPairs {
					require.False(t, pk.IsWhitelistedOracle(ctx, pair.String(), oracle))
				}
				gotOraclesMap := pk.GetOraclesForPairs(ctx, paramsPairs)
				gotOracles := gotOraclesMap[paramsPairs[0]]
				require.EqualValues(t, []sdk.AccAddress(nil), gotOracles)
			},
		},
		{
			name: "multiple oracles whitelisted at different times ",
			test: func() {
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				pk := &nibiruApp.PricefeedKeeper

				paramsPairs := pk.GetParams(ctx).Pairs
				for _, pair := range paramsPairs {
					require.EqualValues(t, []sdk.AccAddress(nil), pk.GetOraclesForPair(ctx, pair.String()))
				}

				oracleA := sample.AccAddress()
				oracleB := sample.AccAddress()

				wantOracles := []sdk.AccAddress{oracleA}
				pk.WhitelistOracles(ctx, wantOracles)
				gotOraclesMap := pk.GetOraclesForPairs(ctx, paramsPairs)
				gotOracles := gotOraclesMap[paramsPairs[0]]
				require.EqualValues(t, wantOracles, gotOracles)
				require.NotContains(t, gotOracles, oracleB)

				wantOracles = []sdk.AccAddress{oracleA, oracleB}
				pk.WhitelistOracles(ctx, wantOracles)
				gotOraclesMap = pk.GetOraclesForPairs(ctx, paramsPairs)
				gotOracles = gotOraclesMap[paramsPairs[0]]
				require.EqualValues(t, wantOracles, gotOracles)
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

func TestAddOracleProposalFromJson(t *testing.T) {
	// NOTE config prefix defaults to cosmos rather than nibi without SetPrefixes,
	// causing a bech32 error
	app.SetPrefixes(app.AccountAddressPrefix) // makes the nibi bech32 prefix valid

	t.Log("load example json as bytes")
	okJSON := sdktestutil.WriteToNewTempFile(t, `
	{
		"title": "Cataclysm-004",
		"description": "Whitelists Delphi to post prices for OHM",
		"oracle": "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
		"pairs": ["uohm:uusd"]
	}	
	`)
	contents, err := ioutil.ReadFile(okJSON.Name())
	assert.NoError(t, err)

	t.Log("Unmarshal json bytes into proposal object")
	encodingConfig := simappparams.MakeTestEncodingConfig()
	proposal := &types.AddOracleProposal{}
	err = encodingConfig.Marshaler.UnmarshalJSON(contents, proposal)
	assert.NoError(t, err)

	t.Log("Check that proposal correctness and validity")
	require.NoError(t, proposal.Validate())
	assert.Equal(t, "Cataclysm-004", proposal.Title)
	assert.Equal(t, "Whitelists Delphi to post prices for OHM", proposal.Description)
	assert.Equal(t, "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl", proposal.Oracle)
	assert.Equal(t, []string{"uohm:uusd"}, proposal.Pairs)
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
			nibiruApp, ctx := testutilapp.NewNibiruApp(true)
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
