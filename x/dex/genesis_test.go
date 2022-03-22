package dex_test

import (
	"testing"

	keepertest "github.com/MatrixDao/dex/testutil/keeper"
	"github.com/MatrixDao/dex/testutil/nullify"
	"github.com/MatrixDao/dex/x/dex"
	"github.com/MatrixDao/dex/x/dex/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.DexKeeper(t)
	dex.InitGenesis(ctx, *k, genesisState)
	got := dex.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
