package cli

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	inflationTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Inflation module subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	inflationTxCmd.AddCommand(
		CmdToggleInflation(),
	)

	return inflationTxCmd
}

func CmdToggleInflation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toggle-inflation [true | false]",
		Args:  cobra.ExactArgs(1),
		Short: "Toggle inflation on or off",
		Long: strings.TrimSpace(`
Toggle inflation on or off.

Requires sudo permissions.

$ nibid tx inflation toggle-inflation true
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgToggleInflation{
				Sender: clientCtx.GetFromAddress().String(),
				Enable: args[0] == "true",
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
