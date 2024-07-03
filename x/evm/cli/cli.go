package cli

import (
	"fmt"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/sudo/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

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
		SendFunTokenToErc20(),
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

// CmdCreateFunTokenFromBankCoin broadcast MsgCreateFunToken
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

// SendFunTokenToErc20 broadcast MsgSendFunTokenToErc20
func SendFunTokenToErc20() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-fun-token-to-erc-20 [to_eth_addr] [coin] [flags]",
		Short: `Send bank [coin] to its erc-20 representation for the user [to_eth_addr]"`,
		Args:  cobra.ExactArgs(2),
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

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}
			msg := &evm.MsgSendFunTokenToErc20{
				Sender:    clientCtx.GetFromAddress().String(),
				BankCoin:  coin,
				ToEthAddr: eth.MustNewHexAddrFromStr(args[1]),
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
