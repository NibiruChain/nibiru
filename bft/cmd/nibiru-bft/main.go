package main

import (
	"os"
	"path/filepath"

	cmd "github.com/NibiruChain/nibiru/v2/bft/cmd/nibiru-bft/commands"
	"github.com/NibiruChain/nibiru/v2/bft/cmd/nibiru-bft/commands/debug"
	cfg "github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/cli"
)

func main() {
	rootCmd := cmd.RootCmd
	rootCmd.AddCommand(
		cmd.GenValidatorCmd,
		cmd.InitFilesCmd,
		cmd.LightCmd,
		cmd.ReplayCmd,
		cmd.ReplayConsoleCmd,
		cmd.ResetAllCmd,
		cmd.ResetPrivValidatorCmd,
		cmd.ResetStateCmd,
		cmd.ShowValidatorCmd,
		cmd.TestnetFilesCmd,
		cmd.ShowNodeIDCmd,
		cmd.GenNodeKeyCmd,
		cmd.VersionCmd,
		cmd.RollbackStateCmd,
		cmd.CompactGoLevelDBCmd,
		debug.DebugCmd,
		cli.NewCompletionCmd(rootCmd, true),
	)

	// NOTE:
	// Users wishing to:
	//	* Use an external signer for their validators
	//	* Supply an in-proc abci app
	//	* Supply a genesis doc file from another source
	//	* Provide their own DB implementation
	// can copy this file and use something other than the
	// DefaultNewNode function
	//
	// Nibiru BFT: newNode (apps.go) wraps DefaultNewNode's setup and adds
	// the built-in "kvstore", "persistent_kvstore", and "e2e" apps.
	nodeFunc := newNode

	// Create & start node
	rootCmd.AddCommand(cmd.NewRunNodeCmd(nodeFunc))

	cmd := cli.PrepareBaseCmd(rootCmd, "CMT", os.ExpandEnv(filepath.Join("$HOME", cfg.DefaultTendermintDir)))
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
