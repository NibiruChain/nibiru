package dex_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/x/testutil/testapp"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/dex"
	"github.com/NibiruChain/nibiru/x/dex/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	app, ctx := testapp.NewTestNibiruAppAndContext(true)
	dex.InitGenesis(ctx, app.DexKeeper, genesisState)
	got := dex.ExportGenesis(ctx, app.DexKeeper)
	require.NotNil(t, got)

	testutil.Fill(&genesisState)
	testutil.Fill(got)

	require.Equal(t, genesisState, *got)
}
