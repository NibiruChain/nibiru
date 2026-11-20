package main

import (
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/x/cli/debugcmd"
)

// GetBuildWasmMsg returns the legacy top-level `build-stargate-wasm` command.
// It is hidden and shares its implementation, and output, with
// `nibid debug stargate`. Boku's msig-ops tooling still calls it.
func GetBuildWasmMsg() *cobra.Command {
	return debugcmd.NewStargateCmd("build-stargate-wasm [message data]", true)
}
