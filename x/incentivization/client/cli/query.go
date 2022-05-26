package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/incentivization/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  types.ModuleName,
		RunE: client.ValidateCmd,
	}

	cmd.AddCommand(
		GetQueryProgramCmd(),
		GetQueryProgramsCmd(),
	)

	return cmd
}

func GetQueryProgramsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "programs [offset] [limit]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			offset, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			limit, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			reverse, err := cmd.Flags().GetBool("reverse")
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.IncentivizationPrograms(cmd.Context(), &types.QueryIncentivizationProgramsRequest{Pagination: &query.PageRequest{
				Offset:  offset,
				Limit:   limit,
				Reverse: reverse,
			}})

			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool("reverse", false, "reverse iteration")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetQueryProgramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "program [id]",
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

			queryClient := types.NewQueryClient(clientCtx)
			resp, err := queryClient.IncentivizationProgram(cmd.Context(), &types.QueryIncentivizationProgramRequest{Id: id})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(resp)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
