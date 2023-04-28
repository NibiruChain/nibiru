package wasmbin

import (
	"os"
	"strings"
)

// WasmKey is an enum type for the available module binding contracts.
type WasmKey int

const (
	WasmKeyPerpBinding WasmKey = iota
	// WasmKeyEpochsBinding // for example...
)

// ToPath Returns a relative file path to compile Wasm bytecode. Wasm bytecode refers
// to compiled WebAssembly binary format that can be executed on the Wasm VM.
func (wasmKey WasmKey) ToPath(pathToWasmbin string) string {
	wasmFileName := WasmBzMap[wasmKey]
	return strings.Join([]string{pathToWasmbin, wasmFileName}, "/")
}

// ToByteCode Returns the Wasm bytecode corresponding to the WasmKey. This can be stored
// directly with the WasmKeeper.
func (wasmKey WasmKey) ToByteCode(pathToWasmbin string) (wasmBytecode []byte, err error) {
	return os.ReadFile(wasmKey.ToPath(pathToWasmbin))
}

// WasmBzMap is a map from WasmKey to the filename for its wasm bytecode.
var WasmBzMap = map[WasmKey]string{
	WasmKeyPerpBinding: "bindings_perp.wasm",
}
