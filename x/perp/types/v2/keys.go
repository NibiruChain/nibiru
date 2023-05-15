package v2

const (
	ModuleName           = "v2perp"
	VaultModuleAccount   = "perp_v2_vault"
	PerpEFModuleAccount  = "perp_v2_ef"
	FeePoolModuleAccount = "perp_v2_fee_pool"
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
