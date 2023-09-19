package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

const FlagVersioned = "versioned"

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
		CmdQueryPosition(),
		CmdQueryPositions(),
		CmdQueryModuleAccounts(),
		CmdQueryMarkets(),
	}
	for _, cmd := range cmds {
		moduleQueryCmd.AddCommand(cmd)
	}

	return moduleQueryCmd
}

// sample token-pair: btc:nusd
func CmdQueryPosition() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [trader] [token-pair]",
		Short: "trader's position for a given token pair/perp/amm",
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

func CmdQueryModuleAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module-accounts",
		Short: "shows all the module accounts in the blockchain",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ModuleAccounts(cmd.Context(), &types.QueryModuleAccountsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryMarkets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "markets",
		Short: "Query all market info",
		Long: `
Query all market info. By default, only active tradable markets are shown.
If --versioned is to to true, the query will return all markets including the 
inactive ones.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			versioned, err := cmd.Flags().GetBool(FlagVersioned)
			if err != nil {
				return err
			}

			res, err := queryClient.QueryMarkets(cmd.Context(), &types.QueryMarketsRequest{
				Versioned: versioned,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(FlagVersioned, false, "toggles whether to include inactive markets")

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
