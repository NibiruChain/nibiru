package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// NewTxCmd returns the transaction commands for this module
func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdCreateDenom(),
		CmdChangeAdmin(),
		CmdMint(),
		CmdBurn(),
		// CmdModifyDenomMetadata(),
	)

	return cmd
}

// CmdCreateDenom broadcast MsgCreateDenom
func CmdCreateDenom() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-denom [subdenom] [flags]",
		Short: "create a new denom from an account",
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

			txFactory = txFactory.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(
				clientCtx.AccountRetriever)

			msg := &types.MsgCreateDenom{
				Sender:   clientCtx.GetFromAddress().String(),
				Subdenom: args[0],
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdChangeAdmin: Broadcasts MsgChangeAdmin
func CmdChangeAdmin() *cobra.Command {
	cmd := &cobra.Command{
		Use: "change-admin [denom] [new-admin-address] [flags]",
		Short: strings.Join([]string{
			"Changes the admin address for a token factory denom.",
			"Must have admin authority to do so."}, " "),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txFactory, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txFactory = txFactory.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			msg := &types.MsgChangeAdmin{
				Sender:   clientCtx.GetFromAddress().String(),
				Denom:    args[0],
				NewAdmin: args[1],
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdMint: Broadcast MsgMint
func CmdMint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [coin] [mint-to] [flags]",
		Short: "Mint a denom to an address. Tx signer must be the denom admin.",
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

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			mintTo, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}
			msg := &types.MsgMint{
				Sender: clientCtx.GetFromAddress().String(),
				Coin:   coin,
				MintTo: mintTo.String(),
			}

			txFactory = txFactory.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdBurn: Broadcast MsgBurn
func CmdBurn() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn [coin] [burn-from] [flags]",
		Short: "Burn tokens from an address. Must have admin authority to do so.",
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
			txFactory = txFactory.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			burnFrom, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}
			msg := &types.MsgBurn{
				Sender:   clientCtx.GetFromAddress().String(),
				Coin:     coin,
				BurnFrom: burnFrom.String(),
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txFactory, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
