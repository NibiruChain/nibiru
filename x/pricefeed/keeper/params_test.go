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

func TestWhitelistOracles(t *testing.T) {
	var noOracles []sdk.AccAddress

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
				for _, pair := range pk.GetPairs(ctx) {
					require.NotContains(t, pair.Oracles, oracle)
					require.EqualValues(t, pair.Oracles, noOracles)
				}
				require.EqualValues(t,
					pk.GetAuthorizedAddresses(ctx), noOracles)
			},
		},
		{
			name: "multiple oracles whitelisted at different times ",
			test: func() {
				nibiruApp, ctx := testutilapp.NewNibiruApp(true)
				pk := &nibiruApp.PricefeedKeeper

				for _, pair := range pk.GetPairs(ctx) {
					require.EqualValues(t, pair.Oracles, noOracles)
				}
				require.EqualValues(t,
					pk.GetAuthorizedAddresses(ctx), noOracles)

				oracleA := sample.AccAddress()
				oracleB := sample.AccAddress()

				wantOracles := []sdk.AccAddress{oracleA}
				pk.WhitelistOracles(ctx, wantOracles)
				gotOracles := pk.GetAuthorizedAddresses(ctx)
				require.EqualValues(t, wantOracles, gotOracles)
				require.NotContains(t, gotOracles, oracleB)

				wantOracles = []sdk.AccAddress{oracleA, oracleB}
				pk.WhitelistOracles(ctx, wantOracles)
				gotOracles = pk.GetAuthorizedAddresses(ctx)
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
	// NOTE There's odd behavior where the test passes when you run the everything
	// but fails when run individually due to a bech32 prefix error.
	app.SetPrefixes(app.AccountAddressPrefix) // makes the nibi bech32 prefix valid

	t.Log("load example json as bytes")
	okJSON := sdktestutil.WriteToNewTempFile(t, `
	{
		"title": "Cataclysm-004",
		"description": "Whitelists Delphi to post prices for OHM",
		"oracle": "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
		"pairs": ["uohm:uusd"],
		"deposit": "1000unibi"
	}	
	`)
	contents, err := ioutil.ReadFile(okJSON.Name())
	assert.NoError(t, err)

	t.Log("Unmarshal json bytes into proposal object")
	encodingConfig := simappparams.MakeTestEncodingConfig()
	proposal := &types.AddOracleProposalWithDeposit{}
	err = encodingConfig.Marshaler.UnmarshalJSON(contents, proposal)
	assert.NoError(t, err)

	t.Log("Check that proposal correctness and validity")
	require.NoError(t, proposal.Validate())
	assert.Equal(t, "Cataclysm-004", proposal.Title)
	assert.Equal(t, "Whitelists Delphi to post prices for OHM", proposal.Description)
	assert.Equal(t, "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl", proposal.Oracle)
	assert.Equal(t, []string{"uohm:uusd"}, proposal.Pairs)
	proposalDeposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
	assert.NoError(t, err)
	assert.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1_000)), proposalDeposit)
}
