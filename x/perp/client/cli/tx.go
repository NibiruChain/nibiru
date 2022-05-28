package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Generalized automated market maker transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		RemoveMarginCmd(),
		AddMarginCmd(),
		LiquidateCmd(),
		OpenPositionCmd(),
	)

	return txCmd
}

func OpenPositionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-position [buy/sell] [pair] [leverage] [amount/sdk.Dec] [base asset amount limit/sdk.Dec]",
		Short: "Opens a position",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).
				WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			var side types.Side
			switch args[0] {
			case "buy":
				side = types.Side_BUY
			case "sell":
				side = types.Side_SELL
			default:
				return fmt.Errorf("invalid side: %s", args[0])
			}

			_, err = common.NewAssetPairFromStr(args[1])
			if err != nil {
				return err
			}

			leverage, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return err
			}

			amount, ok := sdk.NewIntFromString(args[3])
			if !ok {
				return fmt.Errorf("invalid quote amount: %s", args[3])
			}

			baseAssetAmountLimit, ok := sdk.NewIntFromString(args[4])
			if !ok {
				return fmt.Errorf("invalid base amount limit: %s", args[3])
			}

			msg := &types.MsgOpenPosition{
				Sender:               clientCtx.GetFromAddress(),
				TokenPair:            args[1],
				Side:                 side,
				QuoteAssetAmount:     amount,
				Leverage:             leverage,
				BaseAssetAmountLimit: baseAssetAmountLimit,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

/*
RemoveMarginCmd is a CLI command that removes margin from a position,
realizing any outstanding funding payments and decreasing the margin ratio.
*/
func RemoveMarginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-margin [vpool] [margin]",
		Short: "Removes margin from a position, decreasing its margin ratio",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
			$ %s tx perp remove-margin osmo:nusd 100nusd
			`, version.AppName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithTxConfig(
				clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			marginToRemove, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgRemoveMargin{
				Sender:    clientCtx.GetFromAddress(),
				TokenPair: args[0],
				Margin:    marginToRemove,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func AddMarginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-margin [vpool] [margin]",
		Short: "Adds margin to a position, increasing its margin ratio",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
			$ %s tx perp add-margin osmo:nusd 100nusd
			`, version.AppName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithTxConfig(
				clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			marginToAdd, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgAddMargin{
				Sender:    clientCtx.GetFromAddress(),
				TokenPair: args[0],
				Margin:    marginToAdd,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func LiquidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "liquidate [vpool] [trader]",
		Short: "liquidates the position of 'trader' on 'vpool' if possible",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
			$ %s tx perp liquidate osmo:nusd nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl
			`, version.AppName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithTxConfig(
				clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			traderAddr, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgLiquidate{
				Sender:    clientCtx.GetFromAddress(),
				TokenPair: args[0],
				Trader:    traderAddr,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
