package perp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

// TestModuleAccounts verifies that all x/perp module accounts are connected
// to the base application
func TestModuleAccounts(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

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
