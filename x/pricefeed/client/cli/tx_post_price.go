package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	tmtime "github.com/tendermint/tendermint/types/time"
)

var _ = strconv.Itoa(0)

func CmdPostPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post-price [market-id] [price] [expiry]",
		Short: "Broadcast message PostPrice",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argMarketId := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			price, err := sdk.NewDecFromStr(args[1])
			if err != nil {
				return err
			}

			expiryInt, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid expiry %s: %w", args[2], err)
			}

			if expiryInt > types.MaxExpiry {
				return fmt.Errorf("invalid expiry; got %d, max: %d", expiryInt, types.MaxExpiry)
			}

			expiry := tmtime.Canonical(time.Unix(expiryInt, 0))

			msg := types.NewMsgPostPrice(
				clientCtx.GetFromAddress().String(),
				argMarketId,
				price,
				expiry,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
