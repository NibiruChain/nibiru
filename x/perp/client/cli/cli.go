package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/NibiruChain/nibiru/x/perp/types"
)

// ---------------------------------------------------------------------------
// QueryCmd
// ---------------------------------------------------------------------------

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group stablecoin queries under a subcommand
	perpQueryCmd := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmds := []*cobra.Command{
		CmdQueryParams(),
	}
	for _, cmd := range cmds {
		perpQueryCmd.AddCommand(cmd)
	}

	return perpQueryCmd
}

func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "shows the parameters of the x/perp module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(
				context.Background(), &types.QueryParamsRequest{},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// ---------------------------------------------------------------------------
// TxCmd
// ---------------------------------------------------------------------------

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
	)

	return txCmd
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
			$ %s tx perp remove-margin osmo-nusd 100nusd
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
				Sender: clientCtx.GetFromAddress().String(),
				Vpool:  args[0],
				Margin: marginToRemove,
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
