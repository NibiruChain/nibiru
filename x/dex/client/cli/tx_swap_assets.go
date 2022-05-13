package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/dex/types"
)

var _ = strconv.Itoa(0)

func CmdSwapAssets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap-assets",
		Short: "swap assets by specifying tokens in and a token out denom",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
Example:
$ %s tx dex swap-assets --pool-id 1 --tokens-in 100stake --token-out-denom validatortoken --from validator
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
