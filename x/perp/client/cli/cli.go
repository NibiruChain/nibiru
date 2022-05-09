package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

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
		FooCmd(),
	)

	return txCmd
}

/*
FooCmd is a CLI command that does foo.
Example: "foo argfoo"
*/
func FooCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "foo [argfoo]",
		Short: "foo description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithTxConfig(
				clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			msg := &types.MsgFoo{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
