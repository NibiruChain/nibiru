package pricefeed_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/pricefeed"
	"github.com/MatrixDao/matrix/x/pricefeed/types"
	"github.com/MatrixDao/matrix/x/testutil/nullify"
	"github.com/MatrixDao/matrix/x/testutil/testkeeper"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := testkeeper.PricefeedKeeper(t)
	pricefeed.InitGenesis(ctx, k, genesisState)
	got := pricefeed.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
