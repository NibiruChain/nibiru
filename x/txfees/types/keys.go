package types

const (
	ModuleName                        = "txfees"
	StoreKey                          = ModuleName
	RouterKey                         = ModuleName
	FeeCollectorName                  = "fee_collector"
	FeeCollectorForStakingRewardsName = "non_native_fee_collector"
)

var (
	BaseDenomKey         = []byte("base_denom")
	FeeTokensStorePrefix = []byte("fee_tokens")
)
