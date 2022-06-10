package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
		CmdGetVpools(),
		CmdGetBaseAssetPrice(),
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

			tokenPair, err := common.NewAssetPairFromStr(args[0])
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

func CmdGetVpools() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-pools",
		Short: "query all pools information",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.AllPools(
				cmd.Context(),
				&types.QueryAllPoolsRequests{},
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

func CmdGetBaseAssetPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prices [pair] [direction] [base-asset-amount]",
		Short: "calls the GetBaseAssetPrice function, direction is 1 (ADD_TO_POOL) or 2 (REMOVE_FROM_POOL)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			tokenPair, err := common.NewAssetPairFromStr(args[0])
			if err != nil {
				return err
			}

			arg1, err := strconv.ParseInt(args[1], 10, 32)
			direction := types.Direction(arg1)
			if err != nil {
				return err
			}

			baseAssetAmount, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return fmt.Errorf("invalid base asset amount %s", args[2])
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.BaseAssetPrice(
				cmd.Context(),
				&types.QueryBaseAssetRequest{
					Pair:            tokenPair.String(),
					Direction:       direction,
					BaseAssetAmount: baseAssetAmount,
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
