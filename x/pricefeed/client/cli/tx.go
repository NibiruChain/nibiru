package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		CmdPostPrice(),
	)

	return txCmd
}

func CmdPostPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post-price [token0] [token1] [price] [expiry]",
		Short: "Broadcast message PostPrice",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			token0, token1 := args[0], args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			price, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return err
			}

			expiryInt, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid expiry %s: %w", args[2], err)
			}

			if expiryInt > types.MaxExpiry {
				return fmt.Errorf("invalid expiry; got %d, max: %d", expiryInt, types.MaxExpiry)
			}

			msg := types.NewMsgPostPrice(
				clientCtx.GetFromAddress().String(),
				token0, token1,
				price,
				tmtime.Canonical(time.Unix(expiryInt, 0)),
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
