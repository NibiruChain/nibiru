package sudo

import "github.com/NibiruChain/nibiru/v2/x/nutil/set"

type RootAction string

const (
	AddContracts    RootAction = "add_contracts"
	RemoveContracts RootAction = "remove_contracts"
	// EditWasmBlockHooksContract configures the optional Wasm contract address
	// that x/wasm can read for ABCI block-hook dispatch.
	EditWasmBlockHooksContract RootAction = "edit_wasm_block_hooks_contract"
)

// RootActions set[string]: The set of all root actions.
var RootActions = set.New[RootAction](
	AddContracts,
	RemoveContracts,
	EditWasmBlockHooksContract,
)
