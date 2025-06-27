package cli

import (
	"fmt"
	"strings"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("x/%s transaction subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmds := []*cobra.Command{
		CmdSetFeeTokens(),
	}
	for _, cmd := range cmds {
		txCmd.AddCommand(cmd)
	}

	return txCmd
}

func CmdSetFeeTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-fee-token [erc20-address]",
		Args:  cobra.ExactArgs(1),
		Short: "Set fee tokens for the chain",
		Long: strings.TrimSpace(`
Set fee tokens for the chain

$ nibid tx txfees set-fee-token [erc20-address]
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSetFeeTokens{
				Sender: clientCtx.GetFromAddress().String(),
				FeeTokens: []types.FeeToken{
					{
						Denom: args[0],
					},
				},
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
