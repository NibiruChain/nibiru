package types

import "github.com/NibiruChain/nibiru/x/common"

const (
	ModuleName           = "perp"
	VaultModuleAccount   = "vault"
	PerpEFModuleAccount  = "perp_ef"
	FeePoolModuleAccount = "fee_pool"
)

var (
	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	MemStoreKey = "mem_" + ModuleName

	// RouterKey is the message route for perp.
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key.
	QuerierRoute = ModuleName
)

var ModuleAccounts = []string{
	ModuleName,
	PerpEFModuleAccount,
	VaultModuleAccount,
	FeePoolModuleAccount,
	common.TreasuryPoolModuleAccount,
}
