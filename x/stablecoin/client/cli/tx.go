package cli

import (
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Generalized automated market maker transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		MintStableCmd(),
		BurnStableCmd(),
	)

	return txCmd
}

/*
MintStableCmd is a CLI command that mints Nibiru stablecoins.
Example: "mint-sc 100unusd"
*/
func MintStableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint-sc [token-in]",
		Short: "Mint Nibiru stablecoin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			msg, err := buildMintStableMsg(clientCtx, args[0])
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
buildMintStableMsg
*/
func buildMintStableMsg(
	clientCtx client.Context, tokenInStr string,
) (sdk.Msg, error) {

	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		return nil, err
	}

	msg := &types.MsgMintStable{
		Creator: clientCtx.GetFromAddress().String(),
		Stable:  tokenIn,
	}

	return msg, nil
}

func BurnStableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn-sc [token-in]",
		Short: "Burn Nibiru stablecoin commands",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf := tx.NewFactoryCLI(clientCtx, cmd.Flags()).
				WithTxConfig(clientCtx.TxConfig).
				WithAccountRetriever(clientCtx.AccountRetriever)

			msg, err := buildBurnStableMsg(clientCtx, args[0])
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func buildBurnStableMsg(
	clientCtx client.Context, tokenInStr string,
) (sdk.Msg, error) {
	tokenIn, err := sdk.ParseCoinNormalized(tokenInStr)
	if err != nil {
		return nil, err
	}

	msg := &types.MsgBurnStable{
		Creator: clientCtx.GetFromAddress().String(),
		Stable:  tokenIn,
	}

	return msg, nil
}
