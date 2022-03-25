package stablecoin_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	keepertest "github.com/MatrixDao/matrix/x/testutil/keeper"
	"github.com/MatrixDao/matrix/x/testutil/nullify"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.StablecoinKeeper(t)
	stablecoin.InitGenesis(ctx, *k, genesisState)
	got := stablecoin.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
