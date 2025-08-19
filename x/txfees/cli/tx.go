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

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	txFeesTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "TxFees module subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txFeesTxCmd.AddCommand(
		CmdAddFeeToken(),
		CmdRemoveFeeToken(),
	)

	return txFeesTxCmd
}

func CmdAddFeeToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-fee-token --token-address [token-address] --pair [pair] --token-type [token-type] ",
		Args:  cobra.ExactArgs(0),
		Short: "Add a fee token to the txfees module",
		Long: strings.TrimSpace(`
Add a fee token to the txfees module.

Requires sudo permissions.

$ nibid tx add-fee-token --token-address 0xabc --pair unibi:uusdc --token-type FEE_TOKEN_TYPE_CONVERTIBLE
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

			if tokenAddr, _ := cmd.Flags().GetString("token-address"); tokenAddr != "" {
				if !gethcommon.IsHexAddress(tokenAddr) {
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid token address %s", tokenAddr)
				}
				msg.FeeToken.Address = tokenAddr
			}

			if pair, _ := cmd.Flags().GetString("pair"); pair != "" {
				// TODO: validation
				msg.FeeToken.Pair = pair
			}

			if tokenType, _ := cmd.Flags().GetString("token-type"); tokenType != "" {
				switch tokenType {
				case types.FeeTokenType_FEE_TOKEN_TYPE_CONVERTIBLE.String():
					msg.FeeToken.TokenType = types.FeeTokenType_FEE_TOKEN_TYPE_CONVERTIBLE
				case types.FeeTokenType_FEE_TOKEN_TYPE_SWAPPABLE.String():
					msg.FeeToken.TokenType = types.FeeTokenType_FEE_TOKEN_TYPE_SWAPPABLE
				default:
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid token type %s", tokenType)
				}
			}

			if poolAddr, _ := cmd.Flags().GetString("pool-address"); poolAddr != "" {
				if !gethcommon.IsHexAddress(poolAddr) {
					return sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid pool address %s", poolAddr)
				}
				msg.FeeToken.PoolAddress = poolAddr
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String("name", "", "the name of the fee token")
	cmd.Flags().String("token-address", "", "the address of the fee token in hex format")
	cmd.Flags().String("pair", "", "the pair of the fee token, e.g. unibi:uusdc")
	cmd.Flags().String("token-type", "", "the type of the fee token, must be one of FEE_TOKEN_TYPE_CONVERTIBLE or FEE_TOKEN_TYPE_SWAPPABLE")
	cmd.Flags().String("pool-address", "", "the address of the pool for the fee token, if applicable")

	return cmd
}

func CmdRemoveFeeToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-fee-token --token-address [token-address]",
		Args:  cobra.ExactArgs(0),
		Short: "Remove a fee token to the txfees module",
		Long: strings.TrimSpace(`
Rmove a fee token to the txfees module.

Requires sudo permissions.

$ nibid tx remove-fee-token --token-address 0xabc
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
				msg.FeeToken.Address = tokenAddr
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
