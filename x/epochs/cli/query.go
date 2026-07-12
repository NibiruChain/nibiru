package cli

import (
	"fmt"
	"strings"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/x/nutil/flags"

	"github.com/NibiruChain/nibiru/v2/x/epochs"
)

// GetQueryCmd returns the cli query commands for this module.
func GetQueryCmd() *cobra.Command {
	// Group epochs queries under a subcommand
	cmd := &cobra.Command{
		Use:                        epochs.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", epochs.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdEpochInfos(),
		GetCmdCurrentEpoch(),
	)

	return cmd
}

// GetCmdEpochInfos provide running epochInfos.
func GetCmdEpochInfos() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epoch-infos",
		Short: "Query running epochInfos",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query running epoch infos.

Example:
$ %s query epochs epoch-infos
`,
				version.AppName,
			),
		),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := epochs.NewQueryClient(clientCtx)

			res, err := queryClient.EpochInfos(cmd.Context(), &epochs.QueryEpochInfosRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdCurrentEpoch provides current epoch by specified identifier.
func GetCmdCurrentEpoch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-epoch",
		Short: "Query current epoch by specified identifier",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query current epoch by specified identifier.

Example:
$ %s query epochs current-epoch day
`,
				version.AppName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := epochs.NewQueryClient(clientCtx)

			res, err := queryClient.CurrentEpoch(cmd.Context(), &epochs.QueryCurrentEpochRequest{
				Identifier: args[0],
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
