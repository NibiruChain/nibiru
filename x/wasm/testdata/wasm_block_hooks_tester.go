package testdata

import _ "embed"

// WasmBlockHooksTesterContractWasm is a fixture contract used to exercise
// ABCI hook registry planning and target sudo dispatch in x/wasm tests.
//
//go:embed wasm_block_hooks_tester.wasm
var WasmBlockHooksTesterContractWasm []byte
