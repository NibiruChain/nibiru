package pricefeed_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/pricefeed"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestGenesis_DefaultGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := testutilkeeper.PricefeedKeeper(t)
	pricefeed.InitGenesis(ctx, k, genesisState)
	got := pricefeed.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	assert.EqualValues(t, types.DefaultPairs, got.Params.Pairs)
	assert.Empty(t, got.GenesisOracles)
	assert.Empty(t, got.PostedPrices)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState, *got)
}

func TestGenesis_TestGenesis(t *testing.T) {
	app.SetPrefixes(app.AccountAddressPrefix)
	appGenesis := testapp.NewTestGenesisStateFromDefault()
	pfGenesisState := testapp.GenesisPricefeed()

	nibiruApp := testapp.NewTestAppWithGenesis(appGenesis)
	ctx := nibiruApp.NewContext(false, tmproto.Header{})
	k := nibiruApp.PricefeedKeeper
	pricefeed.InitGenesis(ctx, k, *pfGenesisState)
	params := k.GetParams(ctx)
	assert.Equal(t, pfGenesisState.Params, params)

	// genOracle should be whitelisted on all pairs
	for _, pair := range params.Pairs {
		assert.True(t, k.IsWhitelistedOracle(ctx, pair.String(), pfGenesisState.GenesisOracles[0]))
	}

	// prices are only posted for PairGovStable and PairCollStable
	assert.NotEmpty(t, k.GetRawPrices(ctx, params.Pairs[0].String()))
	assert.NotEmpty(t, k.GetRawPrices(ctx, params.Pairs[1].String()))
}
