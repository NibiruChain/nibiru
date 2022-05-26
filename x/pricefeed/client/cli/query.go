package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group pricefeed queries as subcommands of query
	queryCmd := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	commands := []*cobra.Command{
		CmdQueryParams(),
		CmdPrice(),
		CmdPrices(),
		CmdRawPrices(),
		CmdOracles(),
		CmdPairs(),
	}

	for _, command := range commands {
		queryCmd.AddCommand(command)
	}

	return queryCmd
}

func CmdPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price [pair-id]",
		Short: "Display current price for the given pair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			pair := args[0]
			params := &types.QueryPriceRequest{PairId: pair}

			res, err := queryClient.Price(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdPrices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prices",
		Short: "Display current prices for all pairs",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryPricesRequest{}

			res, err := queryClient.Prices(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
