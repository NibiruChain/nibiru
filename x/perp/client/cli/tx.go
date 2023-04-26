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

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
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
		OpenPositionCmd(),
		ClosePositionCmd(),
		MultiLiquidateCmd(),
		DonateToEcosystemFundCmd(),
		EditPegMultiplierCmd(),
	)

	return txCmd
}

func MultiLiquidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-liquidate [Pair1:Trader1] [Pair2:Trader2] ...",
		Short: "liquidates multiple positions at once",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
			$ %s tx perp multi-liquidate ubtc:unusd:nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl ueth:unusd:nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl
			`, version.AppName),
		),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			liquidations := make([]*types.MsgMultiLiquidate_Liquidation, len(args))

			for i, arg := range args {
				parts := strings.Split(arg, ":")
				if len(parts) != 3 {
					return fmt.Errorf("invalid liquidation format: %s", arg)
				}

				pair, err := asset.TryNewPair(fmt.Sprintf("%s:%s", parts[0], parts[1]))
				if err != nil {
					return err
				}

				traderAddr, err := sdk.AccAddressFromBech32(parts[2])
				if err != nil {
					return err
				}

				liquidations[i] = &types.MsgMultiLiquidate_Liquidation{
					Pair:   pair,
					Trader: traderAddr.String(),
				}
			}

			msg := &types.MsgMultiLiquidate{
				Sender:       clientCtx.GetFromAddress().String(),
				Liquidations: liquidations,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func OpenPositionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-position [buy/sell] [pair] [leverage] [quoteAmt / sdk.Dec] [baseAmtLimit / sdk.Dec]",
		Short: "Opens a position",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var side perpammtypes.Direction
			switch args[0] {
			case "buy":
				side = perpammtypes.Direction_LONG
			case "sell":
				side = perpammtypes.Direction_SHORT
			default:
				return fmt.Errorf("invalid side: %s", args[0])
			}

			assetPair, err := asset.TryNewPair(args[1])
			if err != nil {
				return err
			}

			leverage := sdk.MustNewDecFromStr(args[2])

			amount, ok := sdk.NewIntFromString(args[3])
			if !ok {
				return fmt.Errorf("invalid quote amount: %s", args[3])
			}

			baseAmtLimit := sdk.MustNewDecFromStr(args[4])

			msg := &types.MsgOpenPosition{
				Sender:               clientCtx.GetFromAddress().String(),
				Pair:                 assetPair,
				Side:                 side,
				QuoteAssetAmount:     amount,
				Leverage:             leverage,
				BaseAssetAmountLimit: baseAmtLimit.RoundInt(),
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

// TODO: how is a position idenitfiied? by pair? by id?
func ClosePositionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close-position [pair]",
		Short: "Closes a position",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pair, err := asset.TryNewPair(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgClosePosition{
				Sender: clientCtx.GetFromAddress().String(),
				Pair:   pair,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
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
		Use:   "remove-margin [market] [margin]",
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

			marginToRemove, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			pair, err := asset.TryNewPair(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgRemoveMargin{
				Sender: clientCtx.GetFromAddress().String(),
				Pair:   pair,
				Margin: marginToRemove,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func AddMarginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-margin [market] [margin]",
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

			marginToAdd, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			pair, err := asset.TryNewPair(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgAddMargin{
				Sender: clientCtx.GetFromAddress().String(),
				Pair:   pair,
				Margin: marginToAdd,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func DonateToEcosystemFundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "donate-ef [amount]",
		Short: "Donates <amount> of coins to the Ecosystem Fund.",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
			$ %s tx perp donate-ef 100unusd
			`, version.AppName),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			donation, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgDonateToEcosystemFund{
				Sender:   clientCtx.GetFromAddress().String(),
				Donation: donation,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func EditPegMultiplierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repeg [pair] [multiplier]",
		Short: "Repeg the pair price multiplier to the given multiplier",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
			$ %s tx perp repeg ubtc:unusd 30000
			`, version.AppName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			parts := strings.Split(args[0], ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid pair format: %s", args[0])
			}

			pair, err := asset.TryNewPair(fmt.Sprintf("%s:%s", parts[0], parts[1]))
			if err != nil {
				return err
			}

			msg := &types.MsgEditPoolPegMultiplier{
				Sender:        clientCtx.GetFromAddress().String(),
				Pair:          pair,
				PegMultiplier: sdk.MustNewDecFromStr(args[1]),
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
