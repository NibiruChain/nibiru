package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

			poolFile, err := cmd.Flags().GetString(FlagPoolFile)
			if err != nil {
				return err
			}
			if poolFile == "" {
				return fmt.Errorf("must pass in a pool json using the --%s flag", FlagPoolFile)
			}

			contents, err := ioutil.ReadFile(poolFile)
			if err != nil {
				return err
			}

			// make exception if unknown field exists
			pool := &createPoolInputs{}
			if err = json.Unmarshal(contents, pool); err != nil {
				return err
			}

			initialDepositCoins, err := sdk.ParseCoinsNormalized(pool.InitialDeposit)
			if err != nil {
				return err
			}

			poolWeights, err := sdk.ParseDecCoins(pool.Weights)
			if err != nil {
				return err
			}

			if len(initialDepositCoins) != len(poolWeights) {
				return errors.New("deposit tokens and token weights should have same length")
			}

			poolAssets := make([]types.PoolAsset, len(poolWeights))
			for i := 0; i < len(poolWeights); i++ {
				if poolWeights[i].Denom != initialDepositCoins[i].Denom {
					return errors.New("deposit tokens and token weights should have same denom order")
				}

				poolAssets[i] = types.PoolAsset{
					Token:  initialDepositCoins[i],
					Weight: poolWeights[i].Amount.RoundInt(),
				}
			}

			msg := types.NewMsgCreatePool(
				clientCtx.GetFromAddress().String(),
				poolAssets,
				&types.PoolParams{
					SwapFee: sdk.MustNewDecFromStr(pool.SwapFee),
					ExitFee: sdk.MustNewDecFromStr(pool.ExitFee),
				},
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
