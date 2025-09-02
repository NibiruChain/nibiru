package cli

import (
	"strings"

	sdkioerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/spf13/cobra"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	gasTokenTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "GasToken module subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	gasTokenTxCmd.AddCommand(
		CmdAddFeeToken(),
		CmdRemoveFeeToken(),
		CmdUpdateParams(),
	)

	return gasTokenTxCmd
}

func CmdAddFeeToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-fee-token --name [name] --erc20-address [erc20-address]",
		Args:  cobra.ExactArgs(0),
		Short: "Add a fee token to the gastoken module",
		Long: strings.TrimSpace(`
Add a fee token to the gastoken module.

Requires sudo permissions.

$ nibid tx gastoken add-fee-token --name USDC --erc20-address 0xabc
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateFeeToken{
				FeeToken: &types.FeeToken{},
				Sender:   clientCtx.GetFromAddress().String(),
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD,
			}

			if name, _ := cmd.Flags().GetString("name"); name != "" {
				msg.FeeToken.Name = name
			}

			if tokenAddr, _ := cmd.Flags().GetString("erc20-address"); tokenAddr != "" {
				if !gethcommon.IsHexAddress(tokenAddr) {
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid erc20 address %s", tokenAddr)
				}
				msg.FeeToken.Erc20Address = tokenAddr
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String("name", "", "the name of the fee token")
	cmd.Flags().String("erc20-address", "", "the address of the fee token in hex format")

	return cmd
}

func CmdRemoveFeeToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-fee-token --token-address [token-address]",
		Args:  cobra.ExactArgs(0),
		Short: "Remove a fee token from the gastoken module",
		Long: strings.TrimSpace(`
Remove a fee token from the gastoken module.

Requires sudo permissions.

$ nibid tx gastoken remove-fee-token --token-address 0xabc
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateFeeToken{
				FeeToken: &types.FeeToken{},
				Sender:   clientCtx.GetFromAddress().String(),
				Action:   types.FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE,
			}

			if tokenAddr, _ := cmd.Flags().GetString("token-address"); tokenAddr != "" {
				if !gethcommon.IsHexAddress(tokenAddr) {
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid token address %s", tokenAddr)
				}
				msg.FeeToken.Erc20Address = tokenAddr
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String("token-address", "", "the address of the fee token in hex format")

	return cmd
}

func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params --router-address [router-address] --wnibi-address [wnibi-address]",
		Args:  cobra.ExactArgs(0),
		Short: "Update the parameters of the gastoken module.",
		Long: strings.TrimSpace(`
Update the parameters of the gastoken module.

Requires sudo permissions.

$ nibid tx gastoken update-params --router-address 0xrouter --wnibi-address 0xwnibi
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateParams{
				Params: types.Params{},
				Sender: clientCtx.GetFromAddress().String(),
			}

			if routerAddr, _ := cmd.Flags().GetString("router-address"); routerAddr != "" {
				if !gethcommon.IsHexAddress(routerAddr) {
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid router address %s", routerAddr)
				}
				msg.Params.UniswapV3SwapRouterAddress = routerAddr
			}

			if quoterAddr, _ := cmd.Flags().GetString("quoter-address"); quoterAddr != "" {
				if !gethcommon.IsHexAddress(quoterAddr) {
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid quoter address %s", quoterAddr)
				}
				msg.Params.UniswapV3QuoterAddress = quoterAddr
			}

			if wnibiAddr, _ := cmd.Flags().GetString("wnibi-address"); wnibiAddr != "" {
				if !gethcommon.IsHexAddress(wnibiAddr) {
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid wnibi address %s", wnibiAddr)
				}
				msg.Params.WnibiAddress = wnibiAddr
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().String("router-address", "", "the address of the router in hex format")
	cmd.Flags().String("quoter-address", "", "the address of the quoter in hex format")
	cmd.Flags().String("wnibi-address", "", "the address of the wnibi token in hex format")

	return cmd
}
