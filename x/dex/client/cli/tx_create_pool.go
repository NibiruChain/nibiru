package cli

import (
	"strconv"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdCreatePool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [token1] [token2]",
		Short: "Create a pool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			token1, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			token2, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			poolAssets := []types.PoolAsset{}
			poolAssets = append(poolAssets, types.PoolAsset{
				Token:  token1,
				Weight: sdk.NewInt(token1.Amount.Int64()),
			})
			poolAssets = append(poolAssets, types.PoolAsset{
				Token:  token2,
				Weight: sdk.NewInt(token2.Amount.Int64()),
			})

			poolParams := &types.PoolParams{
				SwapFee: sdk.NewDecWithPrec(3, 2),
				ExitFee: sdk.NewDecWithPrec(3, 2),
			}

			msg := types.NewMsgCreatePool(
				clientCtx.GetFromAddress().String(),
				poolAssets,
				poolParams,
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
