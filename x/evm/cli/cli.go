package cli

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/sudo/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/spf13/cobra"
)

// GetTxCmd returns a cli command for this module's transactions
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        evm.ModuleName,
		Short:                      fmt.Sprintf("x/%s transaction subcommands", evm.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmds := []*cobra.Command{
		CmdCreateFunTokenFromBankCoin(),
	}
	for _, cmd := range cmds {
		txCmd.AddCommand(cmd)
	}

	return txCmd
}

// GetQueryCmd returns a cli command for this module's queries
func GetQueryCmd() *cobra.Command {
	moduleQueryCmd := &cobra.Command{
		Use: evm.ModuleName,
		Short: fmt.Sprintf(
			"Query commands for the x/%s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add subcommands
	cmds := []*cobra.Command{}
	for _, cmd := range cmds {
		moduleQueryCmd.AddCommand(cmd)
	}

	return moduleQueryCmd
}

// CmdCreateFunTokenFromBankCoin broadcast MsgCreateFunToken from bank denom
func CmdCreateFunTokenFromBankCoin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-fun-token-from-bank-coin [denom] [flags]",
		Short: `Create an erc-20 fungible token from bank coin [denom]"`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txFactory, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			txFactory = txFactory.
				WithTxConfig(clientCtx.TxConfig).
				WithAccountRetriever(clientCtx.AccountRetriever)

			msg := &evm.MsgCreateFunToken{
				Sender:        clientCtx.GetFromAddress().String(),
				FromBankDenom: args[0],
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
