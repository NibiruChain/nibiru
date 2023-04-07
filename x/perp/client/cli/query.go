package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammcli "github.com/NibiruChain/nibiru/x/perp/amm/cli"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group stablecoin queries under a subcommand
	moduleQueryCmd := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmds := []*cobra.Command{
		CmdQueryParams(),
		CmdQueryPosition(),
		CmdQueryPositions(),
		CmdQueryCumulativePremiumFraction(),
		CmdQueryMetrics(),
		perpammcli.CmdGetVpoolReserveAssets(),
		perpammcli.CmdGetVpools(),
		perpammcli.CmdGetBaseAssetPrice(),
	}
	for _, cmd := range cmds {
		moduleQueryCmd.AddCommand(cmd)
	}

	return moduleQueryCmd
}

func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "shows the parameters of the x/perp module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(
				cmd.Context(), &types.QueryParamsRequest{},
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

// sample token-pair: btc:nusd
func CmdQueryPosition() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [trader] [token-pair]",
		Short: "trader's position for a given token pair/vpool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			trader, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return fmt.Errorf("invalid trader address: %w", err)
			}

			pair, err := asset.TryNewPair(args[1])
			if err != nil {
				return err
			}

			res, err := queryClient.QueryPosition(
				cmd.Context(), &types.QueryPositionRequest{
					Trader: trader.String(),
					Pair:   pair,
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

func CmdQueryPositions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "positions [trader]",
		Short: "return all of a trader's open positions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			trader, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return fmt.Errorf("invalid trader address: %w", err)
			}

			res, err := queryClient.QueryPositions(
				cmd.Context(), &types.QueryPositionsRequest{
					Trader: trader.String(),
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

// sample token-pair: btc:nusd
func CmdQueryCumulativePremiumFraction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "funding-rates [token-pair]",
		Short: "the cumulative funding premium fraction for a market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			assetPair, err := asset.TryNewPair(args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.CumulativePremiumFraction(
				cmd.Context(),
				&types.QueryCumulativePremiumFractionRequest{
					Pair: assetPair,
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

// sample token-pair: btc:nusd
func CmdQueryMetrics() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics [token-pair]",
		Short: "list of perp metrics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			tokenPair, err := asset.TryNewPair(args[0])
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Metrics(
				cmd.Context(),
				&types.QueryMetricsRequest{
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
