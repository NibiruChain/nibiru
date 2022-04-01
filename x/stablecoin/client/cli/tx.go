package cli

import (
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Generalized automated market maker transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewMintStableCmd(),
		NewBurnStableCmd(),
	)

	return txCmd
}

func NewMintStableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [token-in]",
		Short: "mint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			txf, msg, err := NewBuildMintMsg(clientCtx, args[0], txf, cmd.Flags())
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

/*
NewBuildMintMsg
*/
func NewBuildMintMsg(
	clientCtx client.Context, tokenInStr string, txf tx.Factory, fs *flag.FlagSet,
) (tx.Factory, sdk.Msg, error) {

	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		return txf, nil, err
	}

	msg := &types.MsgMintStable{
		Creator: clientCtx.GetFromAddress().String(),
		Stable:  tokenIn,
	}

	return txf, msg, nil
}

func NewBurnStableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn [token-in]",
		Short: "burn",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			txf, msg, err := NewBuildBurnMsg(clientCtx, args[0], txf, cmd.Flags())
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewBuildBurnMsg(
	clientCtx client.Context, tokenInStr string, txf tx.Factory, fs *flag.FlagSet,
) (tx.Factory, sdk.Msg, error) {
	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		return txf, nil, err
	}

	msg := &types.MsgBurnStable{
		Creator: clientCtx.GetFromAddress().String(),
		Stable:  tokenIn,
	}

	return txf, msg, nil
}
