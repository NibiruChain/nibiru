package v2

const (
	ModuleName           = "v2perp"
	VaultModuleAccount   = "v2vault"
	PerpEFModuleAccount  = "v2perp_ef"
	FeePoolModuleAccount = "v2fee_pool"
)

var (
	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	MemStoreKey = "mem_v2perp"

	// RouterKey is the message route for perp.
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key.
	QuerierRoute = ModuleName
)
