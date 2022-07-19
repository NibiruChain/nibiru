package cli

import (
	"github.com/NibiruChain/nibiru/x/vpool/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CreatePoolCmd())

	return cmd
}

func CreatePoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [pair] [trade limit ratio/sdk.Dec] [quote asset reserve/sdk.Dec] [base asset reserve/sdk.Dec] [fluctuation limit/sdk.Dec] [max oracle spread/sdk.Dec]",
		Short: "Create a new virtual pool",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			tlr, err := sdk.NewDecFromStr(args[1])
			if err != nil {
				return err
			}

			qar, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return err
			}

			bar, err := sdk.NewDecFromStr(args[3])
			if err != nil {
				return err
			}

			flr, err := sdk.NewDecFromStr(args[4])
			if err != nil {
				return err
			}

			mos, err := sdk.NewDecFromStr(args[5])

			msg := &types.MsgCreatePool{
				Sender:                clientCtx.GetFromAddress().String(),
				Pair:                  args[0],
				TradeLimitRatio:       tlr,
				QuoteAssetReserve:     qar,
				BaseAssetReserve:      bar,
				FluctuationLimitRatio: flr,
				MaxOracleSpreadRatio:  mos,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
