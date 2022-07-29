package pricefeed_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
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
	assert.Empty(t, got.PostedPrices)
	assert.Empty(t, got.GenesisOracles)

	nullify.Fill(genesisState)
	nullify.Fill(got)

	require.Equal(t, *genesisState, *got)
}

func TestInitGenesis(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
	pricefeedKeeper := nibiruApp.PricefeedKeeper
	oracle := sample.AccAddress()

	pricefeedGenesis := types.DefaultGenesis()
	pricefeedGenesis.GenesisOracles = []string{oracle.String()}
	pricefeedGenesis.PostedPrices = []types.PostedPrice{
		{
			PairID: common.PairGovStable.String(),
			Oracle: oracle.String(),
			Price:  sdk.NewDec(10),
			Expiry: time.Now().Add(1 * time.Hour),
		},
		{
			PairID: common.PairCollStable.String(),
			Oracle: oracle.String(),
			Price:  sdk.OneDec(),
			Expiry: time.Now().Add(1 * time.Hour),
		},
	}

	pricefeed.InitGenesis(ctx, pricefeedKeeper, *pricefeedGenesis)

	t.Log("assert params")
	params := pricefeedKeeper.GetParams(ctx)
	assert.Equal(t, pricefeedGenesis.Params, params)

	assert.True(t, pricefeedKeeper.IsWhitelistedOracle(ctx, common.PairGovStable.String(), oracle))

	t.Log("assert raw prices")
	assert.NotEmpty(t, pricefeedKeeper.GetRawPrices(ctx, common.PairGovStable.String()))
	assert.NotEmpty(t, pricefeedKeeper.GetRawPrices(ctx, common.PairCollStable.String()))
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
