package types

import (
	"github.com/NibiruChain/collections"
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
