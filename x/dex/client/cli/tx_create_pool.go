package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdCreatePool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [flags]",
		Short: "create a new pool and provide the liquidity to it",
		Long: strings.TrimSpace(
			fmt.Sprintf(`create a new pool and provide the liquidity to it.
Pool initialization parameters must be provided through a pool JSON file.

Example:
$ %s tx dex create-pool --pool-file="path/to/pool.json" --from validator --keyring-backend test --home data/localnet --chain-id localnet

Where pool.json contains:
{
	"weights": "1unusd,1uust",
	"initial-deposit": "100unusd,100uust",
	"swap-fee": "0.01",
	"exit-fee": "0.01"
}
`,
				version.AppName,
			),
		),
		Args: cobra.ExactArgs(0),
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
				return types.ErrMissingPoolFileFlag
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
				return types.ErrInvalidCreatePoolArgs
			}

			poolAssets := make([]types.PoolAsset, len(poolWeights))
			for i := 0; i < len(poolWeights); i++ {
				if poolWeights[i].Denom != initialDepositCoins[i].Denom {
					return types.ErrInvalidCreatePoolArgs
				}

				poolAssets[i] = types.PoolAsset{
					Token:  initialDepositCoins[i],
					Weight: poolWeights[i].Amount.RoundInt(),
				}
			}

			msg := types.NewMsgCreatePool(
				/*sender=*/ clientCtx.GetFromAddress().String(),
				poolAssets,
				&types.PoolParams{
					SwapFee: sdk.MustNewDecFromStr(pool.SwapFee),
					ExitFee: sdk.MustNewDecFromStr(pool.ExitFee),
				},
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().AddFlagSet(FlagSetCreatePool())
	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(FlagPoolFile)

	return cmd
}
