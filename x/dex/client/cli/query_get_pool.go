package cli

import (
	"strconv"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdGetPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-pool [pool-id]",
		Short: "Get a pool by its ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			poolId, _ := sdk.NewIntFromString(args[0])

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetPoolRequest{
				PoolId: poolId.Uint64(),
			}

			res, err := queryClient.GetPool(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
