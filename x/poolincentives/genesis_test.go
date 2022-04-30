package poolincentives_test

import (
	"testing"

	keepertest "github.com/NibiruChain/nibiru/testutil/keeper"
	"github.com/NibiruChain/nibiru/testutil/nullify"
	"github.com/NibiruChain/nibiru/x/poolincentives"
	"github.com/NibiruChain/nibiru/x/poolincentives/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:	types.DefaultParams(),
		
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.PoolincentivesKeeper(t)
	poolincentives.InitGenesis(ctx, *k, genesisState)
	got := poolincentives.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	

	// this line is used by starport scaffolding # genesis/test/assert
}
