package types

const (
	ModuleName = "txfees"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	BaseDenomKey         = []byte("base_denom")
	FeeTokensStorePrefix = []byte("fee_tokens")
)
