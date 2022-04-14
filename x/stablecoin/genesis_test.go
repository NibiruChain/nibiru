package stablecoin_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/MatrixDao/matrix/x/stablecoin"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/nullify"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:               types.DefaultParams(),
		ModuleAccountBalance: sdk.NewCoin(common.CollDenom, sdk.ZeroInt()),
	}

	matrixApp, ctx := testutil.NewMatrixApp(true)
	k := matrixApp.StablecoinKeeper
	stablecoin.InitGenesis(ctx, k, genesisState)
	got := stablecoin.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
