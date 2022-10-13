package types

import (
	"github.com/NibiruChain/nibiru/x/common"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
)

var ModuleAccounts = []string{
	perptypes.ModuleName,
	perptypes.VaultModuleAccount,
	perptypes.PerpEFModuleAccount,
	perptypes.FeePoolModuleAccount,
	common.TreasuryPoolModuleAccount,
}
