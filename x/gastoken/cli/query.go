package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

// GetQueryCmd returns the cli query commands for this module.
func GetQueryCmd() *cobra.Command {
	moduleQueryCmd := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Query commands for the x/%s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add subcommands
	cmds := []*cobra.Command{
		GetCmdFeeTokens(),
		GetCmdParams(),
	}
	for _, cmd := range cmds {
		moduleQueryCmd.AddCommand(cmd)
	}
	return moduleQueryCmd

}

func GetCmdFeeTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fee-tokens",
		Short: "Query gas_token viable feetoken",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query gas_token viable feetoken.

Examples:
$ %s query %s fee-tokens
`,
				version.AppName, types.ModuleName,
			),
		),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.FeeTokens(cmd.Context(), &types.QueryFeeTokensRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the parameters of the gas_token module",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the parameters of the gas_token module.

Examples:
$ %s query %s params
`,
				version.AppName, types.ModuleName,
			),
		),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.ParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
