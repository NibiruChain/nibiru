package types

import (
	"github.com/NibiruChain/collections"
)

// constants
const (
	// module name
	ModuleName = "tokenfactory"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for message routing
	RouterKey = ModuleName
)

// KVStore key and mutli-index prefixes
// prefix bytes for the fees persistent store
const (
	KeyPrefixDenom collections.Namespace = iota + 1
	KeyPrefixCreator
	KeyPrefixModuleParams
	KeyPrefixDenomAdmin
	KeyPrefixCreatorIndexer
)
