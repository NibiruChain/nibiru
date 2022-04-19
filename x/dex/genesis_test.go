package dex_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/dex"
	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	app, ctx := testutil.NewNibiruApp(true)
	dex.InitGenesis(ctx, app.DexKeeper, genesisState)
	got := dex.ExportGenesis(ctx, app.DexKeeper)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState, *got)
}
