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
prune, rollback, bft (alias: tendermint), testnet, and evm-tx-index.

The old top-level forms, such as "nibid start", still work.`,
		Example: `  nibid node init <moniker> --chain-id <chain-id>
  nibid node start
  nibid node bft show-node-id
  nibid node genesis validate-genesis`,
		RunE: client.ValidateCmd,
	}
	nodeCmd.AddCommand(newCmds()...)
	renameConsensusCmd(nodeCmd)

	for _, legacy := range newCmds() {
		legacy.Hidden = true
		rootCmd.AddCommand(legacy)
	}
	rootCmd.AddCommand(nodeCmd)
}

// renameConsensusCmd makes `bft` the canonical name of the consensus command
// group under `nibid node`, with `tendermint` as its alias. The hidden
// top-level copy keeps the name `tendermint` so legacy invocations such as
// `nibid --home H tendermint show-node-id` resolve unchanged.
func renameConsensusCmd(nodeCmd *cobra.Command) {
	for _, c := range nodeCmd.Commands() {
		if c.Name() == "tendermint" {
			c.Use = "bft"
			c.Aliases = []string{"tendermint"}
		}
	}
}
