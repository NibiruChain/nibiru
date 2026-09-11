package main

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed localnet.sh
var localnetScriptBytes []byte

func LocalnetCmd() *cobra.Command {
	var printScript bool

	cmd := &cobra.Command{
		Use:   "localnet",
		Short: "Run a single-node Nibiru localnet",
		Long: `Print the embedded localnet.sh script for running a single-node Nibiru localnet.

Run "nibid localnet --script | bash" to start localnet.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !printScript {
				return cmd.Help()
			}

			_, err := fmt.Fprint(cmd.OutOrStdout(), string(localnetScriptBytes))
			return err
		},
	}

	cmd.Flags().BoolVar(&printScript, "script", false, "Print the embedded localnet.sh script")

	return cmd
}
