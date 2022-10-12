package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	utiltypes "github.com/NibiruChain/nibiru/x/util/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use: utiltypes.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module", utiltypes.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	for _, cmd := range []*cobra.Command{
		CmdQueryModuleAccounts(),
	} {
		queryCmd.AddCommand(cmd)
	}

	return queryCmd
}

func CmdQueryModuleAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module-accounts",
		Short: "shows all the module accounts in the blockchain",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := utiltypes.NewQueryClient(clientCtx)

			res, err := queryClient.ModuleAccounts(cmd.Context(), &utiltypes.QueryModuleAccountsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
