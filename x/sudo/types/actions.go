package types

import "github.com/NibiruChain/nibiru/x/common/set"

// RootAction is an enum-like struct for each available action type.
var RootAction = struct {
	AddContracts    string
	RemoveContracts string
}{
	AddContracts:    "add_contracts",
	RemoveContracts: "remove_contracts",
}

// RootActions set[string]: The set of all root actions.
var RootActions = set.New(
	RootAction.AddContracts,
	RootAction.RemoveContracts,
)
