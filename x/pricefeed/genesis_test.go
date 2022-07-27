package pricefeed_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestDefaultGenesis(t *testing.T) {
	app.SetPrefixes(app.AccountAddressPrefix)
	genesisState := types.DefaultGenesis()

	k, ctx := testutilkeeper.PricefeedKeeper(t)
	pricefeed.InitGenesis(ctx, k, *genesisState)
	got := pricefeed.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	assert.EqualValues(t, types.DefaultPairs, got.Params.Pairs)
	assert.Contains(t, got.GenesisOracles, "nibi1hk04vteklhmtwe0zpt7023p5zcgu49e5v3atyp")
	assert.Contains(t, got.GenesisOracles, "nibi10hj3gq54uxd9l5d6a7sn4dcvhd0l3wdgt2zvyp")
	assert.Contains(t, got.GenesisOracles, "nibi1r8gjajmlp9tkff0759rmujv568pa7q6v7u4m3z")
	assert.Empty(t, got.PostedPrices)

	nullify.Fill(genesisState)
	nullify.Fill(got)

	require.Equal(t, *genesisState, *got)
}

func TestGenesis_TestGenesis(t *testing.T) {
	appGenesis := testapp.NewTestGenesisStateFromDefault()
	pfGenesisState := testapp.PricefeedGenesis()

	nibiruApp := testapp.NewNibiruAppWithGenesis(appGenesis)
	ctx := nibiruApp.NewContext(false, tmproto.Header{})
	k := nibiruApp.PricefeedKeeper
	pricefeed.InitGenesis(ctx, k, pfGenesisState)
	params := k.GetParams(ctx)
	assert.Equal(t, pfGenesisState.Params, params)

	// genOracle should be whitelisted on all pairs
	for _, pair := range params.Pairs {
		assert.True(t, k.IsWhitelistedOracle(
			ctx, pair.String(), sdk.MustAccAddressFromBech32(pfGenesisState.GenesisOracles[0])))
	}

	// prices are only posted for PairGovStable and PairCollStable
	assert.NotEmpty(t, k.GetRawPrices(ctx, params.Pairs[0].String()))
	assert.NotEmpty(t, k.GetRawPrices(ctx, params.Pairs[1].String()))
}

func TestGenesisState_Validate(t *testing.T) {
	mockPrivKey := tmtypes.NewMockPV()
	pubkey, err := mockPrivKey.GetPubKey()
	require.NoError(t, err)
	addr := sdk.AccAddress(pubkey.Address())
	now := time.Now()

	examplePairs := common.NewAssetPairs("xrp:bnb")
	for _, tc := range []struct {
		desc     string
		genState *types.GenesisState
		err      error
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			err:      nil,
		},
		{
			desc:     "valid genesis state - empty",
			genState: &types.GenesisState{},
			err:      nil,
		},
		{
			desc: "valid genesis state - full",
			genState: types.NewGenesisState(
				types.NewParams(
					/*pairs=*/ examplePairs,
				),
				[]types.PostedPrice{
					types.NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now)},
			),
			err: nil,
		},
		{
			desc: "invalid posted price - pair must be in genesis params",
			genState: types.NewGenesisState(
				types.NewParams(
					/*pairs=*/ common.AssetPairs{},
				),
				[]types.PostedPrice{
					types.NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now)},
			),
			err: fmt.Errorf("must be in the genesis params"),
		},
		{
			desc: "duplicated posted price at same timestamp - invalid",
			genState: types.NewGenesisState(
				types.NewParams(
					/*pairs=*/ common.AssetPairs{},
				),
				[]types.PostedPrice{
					types.NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now),
					types.NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now),
				},
			),
			err: fmt.Errorf("duplicated posted price"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
