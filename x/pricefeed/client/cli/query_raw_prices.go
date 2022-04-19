package cli

import (
	"strconv"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdRawPrices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw-prices [market-id]",
		Short: "Query RawPrices",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryRawPricesRequest{
				PairId: args[0],
			}

			res, err := queryClient.RawPrices(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
