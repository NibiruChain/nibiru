package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	feesQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	feesQueryCmd.AddCommand(
		GetCmdQueryFeeShares(),
		GetCmdQueryFeeShare(),
		GetCmdQueryParams(),
		GetCmdQueryFeeSharesByWithdrawer(),
	)

	return feesQueryCmd
}

// GetCmdQueryFeeShares implements a command to return all registered contracts
// for fee distribution
func GetCmdQueryFeeShares() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contracts [deployer_addr]",
		Short: "Query dev gas contracts registered with a deployer",
		Long:  "Query dev gas contracts registered with a deployer",
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf("%s query %s deployer-contracts <deployer-address>",
			version.AppName, types.ModuleName,
		),

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			deployer := args[0]
			req := &types.QueryFeeSharesRequest{
				Deployer: deployer,
			}
			if err := req.ValidateBasic(); err != nil {
				return err
			}

			res, err := queryClient.FeeShares(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryFeeShare implements a command to return a registered contract for fee
// distribution
func GetCmdQueryFeeShare() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contract [contract_address]",
		Args:    cobra.ExactArgs(1),
		Short:   "Query a registered contract for fee distribution by its bech32 address",
		Long:    "Query a registered contract for fee distribution by its bech32 address",
		Example: fmt.Sprintf("%s query feeshare contract <contract-address>", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryFeeShareRequest{ContractAddress: args[0]}
			if err := req.ValidateBasic(); err != nil {
				return err
			}

			// Query store
			res, err := queryClient.FeeShare(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryParams implements a command to return the current FeeShare
// parameters.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current feeshare module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryParamsRequest{}

			res, err := queryClient.Params(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryFeeSharesByWithdrawer implements a command that returns all
// contracts that have registered for fee distribution with a given withdraw
// address
func GetCmdQueryFeeSharesByWithdrawer() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "withdrawer-contracts [withdraw_address]",
		Args:    cobra.ExactArgs(1),
		Short:   "Query all contracts that have been registered for feeshare distribution with a given withdrawer address",
		Long:    "Query all contracts that have been registered for feeshare distribution with a given withdrawer address",
		Example: fmt.Sprintf("%s query feeshare withdrawer-contracts <withdrawer-address>", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			withdrawReq := &types.QueryFeeSharesByWithdrawerRequest{
				WithdrawerAddress: args[0],
			}

			if err := withdrawReq.ValidateBasic(); err != nil {
				return err
			}

			// Query store
			goCtx := context.Background()
			res, err := queryClient.FeeSharesByWithdrawer(goCtx, withdrawReq)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
