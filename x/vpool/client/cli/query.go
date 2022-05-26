package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

var _ = strconv.Itoa(0)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group dex queries under a subcommand
	queryCommand := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	for _, cmd := range []*cobra.Command{
		CmdGetVpoolReserveAssets(),
	} {
		queryCommand.AddCommand(cmd)
	}

	return queryCommand
}

func CmdGetVpoolReserveAssets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reserve-assets [pair]",
		Short: "query the reserve assets of a pool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			tokenPair, err := common.NewTokenPairFromStr(args[0])
			if err != nil {
				return err
			}

			res, err := queryClient.ReserveAssets(
				cmd.Context(),
				&types.QueryReserveAssetsRequests{
					Pair: tokenPair.String(),
				},
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
