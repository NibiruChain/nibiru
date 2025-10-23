package sudo

import "github.com/NibiruChain/collections"

const (
	ModuleName = "sudo"
	StoreKey   = ModuleName
)

var (
	NamespaceSudoers       collections.Namespace = 1
	NamespaceZeroGasActors collections.Namespace = 2
)
