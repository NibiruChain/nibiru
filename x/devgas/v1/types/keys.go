package types

import (
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// constants
const (
	// module name
	ModuleName = "devgas"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for message routing
	RouterKey = ModuleName
)

// KVStore key and mutli-index prefixes
// prefix bytes for the fees persistent store
const (
	KeyPrefixFeeShare collections.Namespace = iota + 1
	KeyPrefixDeployer
	KeyPrefixWithdrawer
	KeyPrefixParams
)

// GetKeyPrefixDeployer returns the KVStore key prefix for storing
// registered feeshare contract for a deployer
func GetKeyPrefixDeployer(deployerAddress sdk.AccAddress) []byte {
	return append(KeyPrefixDeployer.Prefix(), deployerAddress.Bytes()...)
}

// GetKeyPrefixWithdrawer returns the KVStore key prefix for storing
// registered feeshare contract for a withdrawer
func GetKeyPrefixWithdrawer(withdrawerAddress sdk.AccAddress) []byte {
	return append(KeyPrefixWithdrawer.Prefix(), withdrawerAddress.Bytes()...)
}
