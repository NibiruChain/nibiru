package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/dex/types"
)

var _ = strconv.Itoa(0)

func CmdJoinPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join-pool",
		Short: "join a new pool and provide the liquidity to it",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			flagSet := cmd.Flags()

			poolId, err := flagSet.GetUint64(FlagPoolId)
			if err != nil {
				return err
			}

			tokensInStrs, err := flagSet.GetStringArray(FlagTokensIn)
			if err != nil {
				return err
			}

			tokensIn := sdk.Coins{}
			for i := 0; i < len(tokensInStrs); i++ {
				parsed, err := sdk.ParseCoinsNormalized(tokensInStrs[i])
				if err != nil {
					return err
				}
				tokensIn = tokensIn.Add(parsed...)
			}

			msg := types.NewMsgJoinPool(
				/*sender=*/ clientCtx.GetFromAddress().String(),
				poolId,
				tokensIn,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, flagSet, msg)
		},
	}

	cmd.Flags().AddFlagSet(FlagSetJoinPool())
	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(FlagPoolId)
	_ = cmd.MarkFlagRequired(FlagTokensIn)

	return cmd
}
