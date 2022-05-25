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

	"github.com/NibiruChain/nibiru/x/incentivization/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCreateIncentivizationProgramCmd(),
		GetFundIncentivizationProgramCmd(),
	)

	return cmd
}

func GetFundIncentivizationProgramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "fund [program-id] [coins]",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			programID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			coins, err := sdk.ParseCoinsNormalized(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgFundIncentivizationProgram{
				Sender: clientCtx.GetFromAddress().String(),
				Id:     programID,
				Funds:  coins,
			}

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func GetCreateIncentivizationProgramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create [denom] [min lockup duration] [number of epochs]",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			denom := args[0]

			duration, err := time.ParseDuration(args[1])
			if err != nil {
				return err
			}

			epochs, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}

			var t *time.Time
			startTimeStr, err := cmd.Flags().GetString("start-time")
			if err != nil {
				return err
			}

			if startTimeStr != "" {
				startTime, err := time.Parse(time.RFC3339, startTimeStr)
				if err != nil {
					return err
				}
				t = &startTime
			}

			var coins sdk.Coins
			coinsStr, err := cmd.Flags().GetString("initial-funds")
			if err != nil {
				return err
			}

			if coinsStr != "" {
				coins, err = sdk.ParseCoinsNormalized(coinsStr)
				if err != nil {
					return err
				}
			}

			msg := &types.MsgCreateIncentivizationProgram{
				Sender:            clientCtx.GetFromAddress().String(),
				LpDenom:           denom,
				MinLockupDuration: &duration,
				StartTime:         t,
				Epochs:            epochs,
				InitialFunds:      coins,
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("start-time", "", "start time (RFC3339) of the incentivization program [default: block time]")
	cmd.Flags().String("initial-funds", "", "initial funds to deploy on the incentivization program [default: 0]")

	return cmd
}
