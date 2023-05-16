package types

const (
	ModuleName           = "v1perp"
	VaultModuleAccount   = "v1vault"
	PerpEFModuleAccount  = "v1perp_ef"
	FeePoolModuleAccount = "v1fee_pool"
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
