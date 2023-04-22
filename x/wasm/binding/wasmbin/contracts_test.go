package wasmbin_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/wasm/binding/wasmbin"
)

func TestExpectedBytecodeExists(t *testing.T) {
	wasmKeys := []wasmbin.WasmKey{
		wasmbin.WasmKeyPerpBinding,
	}

	testCases := make(map[wasmbin.WasmKey]string)
	for _, wasmKey := range wasmKeys {
		var testName string = wasmbin.WasmBzMap[wasmKey]
		testCases[wasmKey] = testName
	}

	for wasmKey, testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			pathToWasmbin := wasmbin.GetPackageDir(t)
			pathToWasmBytecode := wasmKey.ToPath(pathToWasmbin)
			_, err := os.Stat(pathToWasmBytecode)
			var errMsg = ""
			if os.IsNotExist(err) {
				fileName := testName
				errMsg = fmt.Sprintf("File %s does not exist\n", fileName)
			}
			assert.NoErrorf(t, err, errMsg)

			_, err = wasmKey.ToByteCode(pathToWasmbin)
			assert.NoError(t, err)
		})
	}
}
