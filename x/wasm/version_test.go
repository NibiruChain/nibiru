package wasm_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	wasmvm "github.com/NibiruChain/nibiru/v2/lib/wasmvm"
	"github.com/NibiruChain/nibiru/v2/x/wasm"
)

func TestCheckLibwasmVersion(t *testing.T) {
	require.NotEmpty(t, wasmvm.ExpectedVersion)
	require.NoError(t, wasm.CheckLibwasmVersion(wasmvm.ExpectedVersion))

	err := wasm.CheckLibwasmVersion("0.0.0")
	require.ErrorContains(t, err, "libwasmversion mismatch")
}

func TestAddModuleInitFlagsChecksLinkedLibwasmvmVersionByDefault(t *testing.T) {
	startCmd := &cobra.Command{}
	wasm.AddModuleInitFlags(startCmd)

	require.NoError(t, startCmd.PreRunE(startCmd, nil))
}
