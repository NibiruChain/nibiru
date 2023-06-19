package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/stablecoin/types"
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
		BuybackCmd(),
		RecollateralizeCmd(),
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

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			txf = txf.WithTxConfig(
				clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			inCoin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgMintStable{
				Creator: clientCtx.GetFromAddress().String(),
				Stable:  inCoin,
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
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

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf.WithTxConfig(clientCtx.TxConfig).
				WithAccountRetriever(clientCtx.AccountRetriever)

			inCoin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgBurnStable{
				Creator: clientCtx.GetFromAddress().String(),
				Stable:  inCoin,
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func BuybackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buyback [token-in]",
		Short: "sell shares to the protocol in exchange for collateral (UST)",
		Long: `A user can call 'buyback' when there's too much collateral in the 
		 protocol according to the target collateral ratio. The user swaps NIBI 
		 for UST at a 0% transaction fee and the protocol burns the NIBI it 
		 buys from the user.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			inCoin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgBuyback{
				Creator: clientCtx.GetFromAddress().String(),
				Gov:     inCoin}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// TODO: test
func RecollateralizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recoll [token-in]",
		Short: "sell UST to the protocol in exchange for bonus value in NIBI",
		Long: `Recollateralize is a function that incentivizes the caller to add up to 
		the amount of collateral needed to reach some target collateral ratio. 
		Recollateralize checks if the USD value of collateral in the protocol is 
		below the required amount defined by the current collateral ratio.
		Nibiru's NUSD stablecoin is taken to be the dollar that determines USD value.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			txf = txf.WithTxConfig(
				clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			inCoin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgRecollateralize{
				Creator: clientCtx.GetFromAddress().String(),
				Coll:    inCoin}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
