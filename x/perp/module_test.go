package perp_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
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

// TestModuleAccounts verifies that all x/perp module accounts are connected
// to the base application
func TestModuleAccounts(t *testing.T) {
	nibiruApp, ctx := testutil.NewNibiruApp(true)

	perpAcc := nibiruApp.PerpKeeper.AccountKeeper.GetModuleAccount(
		ctx, types.ModuleName)
	assert.NotNil(t, perpAcc)

	vaultAcc := nibiruApp.PerpKeeper.AccountKeeper.GetModuleAccount(
		ctx, types.VaultModuleAccount)
	assert.NotNil(t, vaultAcc)

	perpEFAcc := nibiruApp.PerpKeeper.AccountKeeper.GetModuleAccount(
		ctx, types.PerpEFModuleAccount)
	assert.NotNil(t, perpEFAcc)

	feePoolAcc := nibiruApp.PerpKeeper.AccountKeeper.GetModuleAccount(
		ctx, types.FeePoolModuleAccount)
	assert.NotNil(t, feePoolAcc)
}
