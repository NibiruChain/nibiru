package pb

import "github.com/NibiruChain/nibiru/x/common/set"

// ROOT_ACTION is an enum-like struct for each available action type.
var ROOT_ACTION = struct {
	AddContracts    string
	RemoveContracts string
}{
	AddContracts:    "add_contracts",
	RemoveContracts: "remove_contracts",
}

// ROOT_ACTIONS set[string]: The set of all root actions.
var ROOT_ACTIONS = set.New(
	ROOT_ACTION.AddContracts,
	ROOT_ACTION.RemoveContracts,
)
