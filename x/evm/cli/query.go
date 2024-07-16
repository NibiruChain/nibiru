package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/evm"
)

// GetQueryCmd returns a cli command for this module's queries
func GetQueryCmd() *cobra.Command {
	moduleQueryCmd := &cobra.Command{
		Use: evm.ModuleName,
		Short: fmt.Sprintf(
			"Query commands for the x/%s module", evm.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add subcommands
	cmds := []*cobra.Command{
		GetCmdFunToken(),
	}
	for _, cmd := range cmds {
		moduleQueryCmd.AddCommand(cmd)
	}
	return moduleQueryCmd
}

// GetCmdFunToken returns fungible token mapping for either bank coin or erc20 addr
func GetCmdFunToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "funtoken [coin-or-erc20addr]",
		Short: "Query evm fungible token mapping",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query evm fungible token mapping.

Examples:
$ %s query %s get-fun-token ibc/abcdef
$ %s query %s get-fun-token 0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6
`,
				version.AppName, evm.ModuleName,
				version.AppName, evm.ModuleName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := evm.NewQueryClient(clientCtx)

			res, err := queryClient.FunToken(cmd.Context(), &evm.QueryFunTokenRequest{
				Token: args[0],
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
