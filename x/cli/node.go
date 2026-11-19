package main

import (
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
)

// nodeCmdName is the name of the parent command for node operations.
const nodeCmdName = "node"

// addNodeCommands registers the node-operator commands twice:
//
//   - under the `nibid node` parent, the documented form, and
//   - at the top level with Hidden set, so every legacy form (`nibid start`,
//     `nibid tendermint show-node-id`, `nibid --home H init`, ...) keeps
//     working for cosmovisor, cosmopilot, systemd units, and scripts. Hidden
//     commands only disappear from help; they print no deprecation notice.
//
// A cobra command has exactly one parent, so newCmds is called once per
// registration and must return new command objects each time.
func addNodeCommands(rootCmd *cobra.Command, newCmds func() []*cobra.Command) {
	nodeCmd := &cobra.Command{
		Use:   nodeCmdName,
		Short: "Start, initialize, and maintain a node",
		Long: `Commands for running a Nibiru node: start, init, export, genesis,
prune, rollback, tendermint (alias: comet), testnet, and evm-tx-index.

The old top-level forms, such as "nibid start", still work.`,
		RunE: client.ValidateCmd,
	}
	nodeCmd.AddCommand(newCmds()...)

	for _, legacy := range newCmds() {
		legacy.Hidden = true
		rootCmd.AddCommand(legacy)
	}
	rootCmd.AddCommand(nodeCmd)
}
