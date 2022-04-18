package cli

import (
	"fmt"
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

func CmdExitPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit-pool [flags]",
		Short: "exit a pool by burning pool share tokens",
		Long: strings.TrimSpace(
			fmt.Sprintf(`
Example:
$ %s tx dex exit-pool --pool-id 1 --pool-shares-out 100nibiru/pool/1 --from validator --keyring-backend test --home data/localnet --chain-id localnet
`,
				version.AppName,
			),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			poolId, err := cmd.Flags().GetUint64(FlagPoolId)
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
