package types

import "github.com/NibiruChain/nibiru/x/common/set"

type RootAction string

const (
	AddContracts    RootAction = "add_contracts"
	RemoveContracts RootAction = "remove_contracts"
)

// RootActions set[string]: The set of all root actions.
var RootActions = set.New[RootAction](
	AddContracts,
	RemoveContracts,
)
