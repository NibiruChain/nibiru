package wasm_test

import "testing"

func skipUnsupportedNibiruWasmIBCHarness(t *testing.T) {
	t.Helper()
	t.Skip("TODO(x/wasm): enable after the Nibiru Wasm IBC test harness supports custom keeper options and genesis validator setup")
}
