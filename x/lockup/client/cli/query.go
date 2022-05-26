package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/lockup/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: types.ModuleName}
	cmd.AddCommand(
		GetQueryLockCmd(),
		GetQueryLocksByAddressCmd(),
		GetQueryLockedCoinsCmd(),
	)

	return cmd
}

func GetQueryLockedCoinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "locked-coins [address]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			addr := args[0]
			q := types.NewQueryClient(clientCtx)

			lockedCoins, err := q.LockedCoins(cmd.Context(), &types.QueryLockedCoinsRequest{Address: addr})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(lockedCoins)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetQueryLocksByAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "locks-by-address [address]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			addr := args[0]

			var pagination *query.PageRequest
			offset, err := cmd.Flags().GetUint64("offset")
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetUint64("limit")
			if err != nil {
				return err
			}

			if offset != 0 || limit != 0 {
				pagination = &query.PageRequest{
					Offset: offset,
					Limit:  limit,
				}
			}

			q := types.NewQueryClient(clientCtx)

			locks, err := q.LocksByAddress(cmd.Context(), &types.QueryLocksByAddress{
				Address:    addr,
				Pagination: pagination,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(locks)
		},
	}

	cmd.Flags().Uint64("offset", 0, "offset for pagination")
	cmd.Flags().Uint64("limit", 0, "limit for pagination")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetQueryLockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "lock [lock-id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			q := types.NewQueryClient(clientCtx)
			lock, err := q.Lock(cmd.Context(), &types.QueryLockRequest{Id: id})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(lock)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
