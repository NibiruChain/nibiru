package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
)

var _ = strconv.Itoa(0)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group spot queries under a subcommand
	queryCommand := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	for _, cmd := range []*cobra.Command{
		CmdGetMarketReserveAssets(),
		CmdGetMarkets(),
		CmdGetBaseAssetPrice(),
	} {
		queryCommand.AddCommand(cmd)
	}

	return queryCommand
}

func CmdGetMarketReserveAssets() *cobra.Command {
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

			tokenPair, err := asset.TryNewPair(args[0])
			if err != nil {
				return err
			}

			res, err := queryClient.ReserveAssets(
				cmd.Context(),
				&types.QueryReserveAssetsRequest{
					Pair: tokenPair,
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

func CmdGetMarkets() *cobra.Command {
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
				&types.QueryAllPoolsRequest{},
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
		Short: "calls the GetBaseAssetPrice function, direction is add (LONG) or remove (SHORT)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			tokenPair, err := asset.TryNewPair(args[0])
			if err != nil {
				return err
			}

			baseAssetAmount, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return fmt.Errorf("invalid base asset amount %s", args[2])
			}

			var direction types.Direction
			switch strings.TrimSpace(args[1]) {
			case "add":
				direction = types.Direction_LONG
			case "remove":
				direction = types.Direction_SHORT
			default:
				return fmt.Errorf("invalid direction %s", args[1])
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.BaseAssetPrice(
				cmd.Context(),
				&types.QueryBaseAssetPriceRequest{
					Pair:            tokenPair,
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
