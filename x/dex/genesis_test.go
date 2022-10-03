package dex_test

import (
	"github.com/NibiruChain/nibiru/x/testutil"
	"testing"

	"github.com/NibiruChain/nibiru/simapp"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/dex"
	"github.com/NibiruChain/nibiru/x/dex/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	app, ctx := simapp.NewTestNibiruAppAndContext(true)
	dex.InitGenesis(ctx, app.DexKeeper, genesisState)
	got := dex.ExportGenesis(ctx, app.DexKeeper)
	require.NotNil(t, got)

	testutil.Fill(&genesisState)
	testutil.Fill(got)

	require.Equal(t, genesisState, *got)
}
