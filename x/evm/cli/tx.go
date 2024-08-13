package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"

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
		CmdCreateFunTokenFromERC20(),
		SendFunTokenToEvm(),
	}
	for _, cmd := range cmds {
		txCmd.AddCommand(cmd)
	}

	return txCmd
}

// CmdCreateFunTokenFromBankCoin broadcast MsgCreateFunToken
func CmdCreateFunTokenFromBankCoin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-funtoken-from-bank-coin [denom] [flags]",
		Short: `Create an erc20 fungible token from bank coin [denom]"`,
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

// CmdCreateFunTokenFromERC20 broadcast MsgCreateFunToken
func CmdCreateFunTokenFromERC20() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-funtoken-from-erc20 [erc20addr] [flags]",
		Short: `Create a fungible token from erc20 contract [erc20addr]"`,
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
			erc20Addr, err := eth.NewHexAddrFromStr(args[0])
			if err != nil {
				return err
			}
			msg := &evm.MsgCreateFunToken{
				Sender:    clientCtx.GetFromAddress().String(),
				FromErc20: &erc20Addr,
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// SendFunTokenToEvm broadcast MsgSendFunTokenToEvm
func SendFunTokenToEvm() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-funtoken-to-erc20 [to_eth_addr] [coin] [flags]",
		Short: `Send bank [coin] to its erc20 representation for the user [to_eth_addr]"`,
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

			coin, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			msg := &evm.MsgSendFunTokenToEvm{
				Sender:    clientCtx.GetFromAddress().String(),
				BankCoin:  coin,
				ToEthAddr: eth.MustNewHexAddrFromStr(args[0]),
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
