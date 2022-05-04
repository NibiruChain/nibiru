package perp_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp"
	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:               types.DefaultParams(),
		ModuleAccountBalance: sdk.NewCoin(common.GovDenom, sdk.ZeroInt()),
	}

	nibiruApp, ctx := testutil.NewNibiruApp(true)
	perp.InitGenesis(ctx, nibiruApp.PerpKeeper, genesisState)
	exportedGenesisState := perp.ExportGenesis(ctx, nibiruApp.PerpKeeper)
	require.NotNil(t, exportedGenesisState)

	nullify.Fill(&genesisState)
	nullify.Fill(exportedGenesisState)

	require.Equal(t, genesisState, *exportedGenesisState)
}
