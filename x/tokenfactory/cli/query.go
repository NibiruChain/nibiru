package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

// NewQueryCmd returns the cli query commands for this module
func NewQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Aliases:                    []string{"tf"},
		Short:                      fmt.Sprintf("Queries for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryDenoms(),
		CmdQueryModuleParams(),
		CmdQueryDenomInfo(),
	)

	return cmd
}

// CmdQueryDenoms: Queries all TF denoms for a given creator.
func CmdQueryDenoms() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denoms [creator] [flags]",
		Short: "Returns token denoms created by a given creator address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Denoms(
				cmd.Context(), &types.QueryDenomsRequest{
					Creator: args[0],
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

// CmdQueryModuleParams: Queries module params for x/tokenfactory.
func CmdQueryModuleParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params [flags]",
		Short: "Get the params for the x/tokenfactory module",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(
				cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// CmdQueryDenomInfo: Queries the admin and x/bank metadata for a TF denom
func CmdQueryDenomInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom-info [denom] [flags]",
		Short: "Get the admin and x/bank metadata for a denom",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.DenomInfo(
				cmd.Context(),
				&types.QueryDenomInfoRequest{
					Denom: args[0],
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
