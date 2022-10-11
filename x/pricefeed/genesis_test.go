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
	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestGenesis_DefaultGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := pricefeed.PricefeedKeeper(t)
	pricefeed.InitGenesis(ctx, k, genesisState)
	got := pricefeed.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	assert.EqualValues(t, types.DefaultPairs, got.Params.Pairs)
	assert.Empty(t, got.GenesisOracles)
	assert.Empty(t, got.PostedPrices)

	testutil.Fill(&genesisState)
	testutil.Fill(got)

	require.Equal(t, genesisState, *got)
}

func TestGenesis_TestGenesis(t *testing.T) {
	app.SetPrefixes(app.AccountAddressPrefix)
	appGenesis := simapp.NewTestGenesisStateFromDefault()
	pfGenesisState := simapp.PricefeedGenesis()

	nibiruApp := simapp.NewTestNibiruAppWithGenesis(appGenesis)
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

	// export genesis
	genState := pricefeed.ExportGenesis(ctx, k)
	assert.Equal(t, pfGenesisState.GenesisOracles, genState.GenesisOracles)
	assert.Equal(t, pfGenesisState.Params, genState.Params)
	for i, postedPrice := range pfGenesisState.PostedPrices {
		assert.True(t, postedPrice.Equal(genState.PostedPrices[i]))
	}
}

func TestGenesisState_Validate(t *testing.T) {
	mockPrivKey := tmtypes.NewMockPV()
	pubkey, err := mockPrivKey.GetPubKey()
	require.NoError(t, err)
	addr := sdk.AccAddress(pubkey.Address())
	now := time.Now()

	examplePairs := common.NewAssetPairs("xrp:bnb")
	twapLookbackWindow := 15 * time.Minute
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
					/*twapLookbackWindow=*/ twapLookbackWindow,
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
					/*twapLookbackWindow=*/ twapLookbackWindow,
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
					/*twapLookbackWindow=*/ twapLookbackWindow,
				),
				[]types.PostedPrice{
					types.NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now),
					types.NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now),
				},
			),
			err: fmt.Errorf("duplicated posted price"),
		},
		{
			desc: "negative twap lookback window - invalid",
			genState: types.NewGenesisState(
				types.NewParams(
					/*pairs=*/ common.AssetPairs{},
					/*twapLookbackWindow=*/ -10*time.Second,
				),
				[]types.PostedPrice{},
			),
			err: fmt.Errorf("invalid twapLookbackWindow, negative value is not allowed: -10s"),
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
