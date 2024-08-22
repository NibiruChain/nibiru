package cli

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"

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
		CmdCreateFunToken(),
		CmdSendFunTokenToEvm(),
	}
	for _, cmd := range cmds {
		txCmd.AddCommand(cmd)
	}

	return txCmd
}

// CmdCreateFunToken broadcast MsgCreateFunToken
func CmdCreateFunToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-funtoken [flags]",
		Short: `Create a fungible token from erc20 contract [erc20addr]"`,
		Long: heredoc.Doc(`
	Example: Creating a fungible token mapping from bank coin.

	create-funtoken --bank-denom="ibc/..."

	Example: Creating a fungible token mapping from an ERC20.

	create-funtoken --erc20=[erc20-address]
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			bankDenom, _ := cmd.Flags().GetString("bank-denom")
			erc20AddrStr, _ := cmd.Flags().GetString("erc20")

			if (bankDenom == "" && erc20AddrStr == "") ||
				(bankDenom != "" && erc20AddrStr != "") {
				return fmt.Errorf("exactly one of the flags --bank-denom or --erc20 must be specified")
			}

			msg := &evm.MsgCreateFunToken{
				Sender: clientCtx.GetFromAddress().String(),
			}
			if bankDenom != "" {
				if err := sdk.ValidateDenom(bankDenom); err != nil {
					return err
				}
				msg.FromBankDenom = bankDenom
			} else {
				erc20Addr, err := eth.NewHexAddrFromStr(erc20AddrStr)
				if err != nil {
					return err
				}
				msg.FromErc20 = &erc20Addr
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String("bank-denom", "", "The bank denom to create a fungible token from")
	cmd.Flags().String("erc20", "", "The ERC20 address to create a fungible token from")

	return cmd
}

// CmdSendFunTokenToEvm broadcast MsgSendFunTokenToEvm
func CmdSendFunTokenToEvm() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-funtoken-to-erc20 [to_eth_addr] [coin] [flags]",
		Short: `Send bank [coin] to its erc20 representation for the user [to_eth_addr]"`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coin, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			msg := &evm.MsgSendFunTokenToEvm{
				Sender:    clientCtx.GetFromAddress().String(),
				BankCoin:  coin,
				ToEthAddr: eth.MustNewHexAddrFromStr(args[0]),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
