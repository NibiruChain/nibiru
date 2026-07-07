package testdata

import _ "embed"

//go:embed wasm_block_hooks_tester.wasm
// WasmBlockHooksTesterContractWasm is a fixture contract used to exercise
// ABCI hook registry planning and target sudo dispatch in x/wasm tests.
var WasmBlockHooksTesterContractWasm []byte
