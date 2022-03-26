package dex_test

import (
	"testing"

	"github.com/MatrixDao/matrix/testutil/nullify"
	"github.com/MatrixDao/matrix/x/dex"
	"github.com/MatrixDao/matrix/x/dex/testutil"
	"github.com/MatrixDao/matrix/x/dex/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, _, _, ctx, _ := testutil.CreateKeepers(t, storeKey)
	dex.InitGenesis(ctx, k, genesisState)
	got := dex.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState, *got)
}
