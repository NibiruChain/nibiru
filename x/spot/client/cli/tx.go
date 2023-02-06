package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/spot/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdCreatePool(),
		CmdJoinPool(),
		CmdExitPool(),
		CmdSwapAssets(),
	)

	return cmd
}

func CmdSwapAssets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap-assets",
		Short: "swap assets by specifying tokens in and a token out denom",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
Example:
$ %s tx spot swap-assets --pool-id 1 --tokens-in 100stake --token-out-denom validatortoken --from validator
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
			flagSet := cmd.Flags()

			poolId, err := flagSet.GetUint64(FlagPoolId)
			if err != nil {
				return err
			}

			tokenInStr, err := cmd.Flags().GetString(FlagTokenIn)
			if err != nil {
				return err
			}

			tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
			if err != nil {
				return err
			}

			tokenOutDenom, err := cmd.Flags().GetString(FlagTokenOutDenom)
			if err != nil {
				return err
			}

			msg := types.NewMsgSwapAssets(
				clientCtx.GetFromAddress().String(),
				poolId,
				tokenIn,
				tokenOutDenom,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().AddFlagSet(FlagSetSwapAssets())
	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(FlagPoolId)
	_ = cmd.MarkFlagRequired(FlagTokenIn)
	_ = cmd.MarkFlagRequired(FlagTokenOutDenom)

	return cmd
}

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

			useAllCoins, err := flagSet.GetBool(FlagUseAllCoins)
			if err != nil {
				return err
			}

			msg := types.NewMsgJoinPool(
				/*sender=*/ clientCtx.GetFromAddress().String(),
				poolId,
				tokensIn,
				useAllCoins,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, flagSet, msg)
		},
	}

	cmd.Flags().AddFlagSet(FlagSetJoinPool())
	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(FlagPoolId)
	_ = cmd.MarkFlagRequired(FlagTokensIn)
	_ = cmd.MarkFlagRequired(FlagUseAllCoins)

	return cmd
}

func CmdExitPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit-pool",
		Short: "exit a pool by burning pool share tokens",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
Example:
$ %s tx spot exit-pool --pool-id 1 --pool-shares-out 100nibiru/pool/1 --from validator
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
			flagSet := cmd.Flags()

			poolId, err := flagSet.GetUint64(FlagPoolId)
			if err != nil {
				return err
			}

			poolSharesOut, err := cmd.Flags().GetString(FlagPoolSharesOut)
			if err != nil {
				return err
			}

			parsedPoolSharesOut, err := sdk.ParseCoinNormalized(poolSharesOut)
			if err != nil {
				return err
			}

			msg := types.NewMsgExitPool(
				clientCtx.GetFromAddress().String(),
				poolId,
				parsedPoolSharesOut,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().AddFlagSet(FlagSetExitPool())
	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(FlagPoolId)
	_ = cmd.MarkFlagRequired(FlagPoolSharesOut)

	return cmd
}

func CmdCreatePool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [flags]",
		Short: "create a new pool and provide the liquidity to it",
		Long: strings.TrimSpace(
			fmt.Sprintf(`create a new pool and provide the liquidity to it.
Pool initialization parameters must be provided through a pool JSON file.

Example:
$ %s tx spot create-pool --pool-file="path/to/pool.json" --from validator

Where pool.json contains:
{
	"weights": "1unusd,1uusdc",
	"initial-deposit": "100unusd,100uusdc",
	"swap-fee": "0.01",
	"exit-fee": "0.01",
	"pool-type": "balancer", // 'balancer' or 'stableswap'
	"amplification": "10" // Amplification parameter for the stableswap pool
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

			contents, err := os.ReadFile(poolFile)
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

			var poolType types.PoolType
			if pool.PoolType == "balancer" {
				poolType = types.PoolType_BALANCER
			} else if pool.PoolType == "stableswap" {
				poolType = types.PoolType_STABLESWAP
			} else {
				return types.ErrInvalidCreatePoolArgs
			}

			var amplification sdk.Int
			if poolType == types.PoolType_STABLESWAP {
				amplification, err = pool.AmplificationInt()
				if err != nil {
					return err
				}
			}

			msg := types.NewMsgCreatePool(
				/*sender=*/ clientCtx.GetFromAddress().String(),
				poolAssets,
				&types.PoolParams{
					SwapFee:  sdk.MustNewDecFromStr(pool.SwapFee),
					ExitFee:  sdk.MustNewDecFromStr(pool.ExitFee),
					PoolType: poolType,
					A:        amplification,
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
