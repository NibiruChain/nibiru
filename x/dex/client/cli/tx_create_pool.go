package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/MatrixDao/matrix/x/dex/types"
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
	"weights": "1usdm,1ust",
	"initial-deposit": "100usdm,100ust",
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
