// Copyright (c) 2023-2024 Nibi, Inc.
package ante

import (
	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// AnteDecPreventEthereumTxMsgs prevents invalid msg types from being executed
type AnteDecPreventEthereumTxMsgs struct{}

// AnteHandle rejects messages that requires ethereum-specific authentication.
// For example `MsgEthereumTx` requires fee to be deducted in the antehandler in
// order to perform the refund.
func (anteDec AnteDecPreventEthereumTxMsgs) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		if _, ok := msg.(*evm.MsgEthereumTx); ok {
			return ctx, sdkioerrors.Wrapf(
				sdkerrors.ErrInvalidType,
				"MsgEthereumTx needs to be contained within a tx with 'ExtensionOptionsEthereumTx' option",
			)
		}
	}

	if ctx.ChainID() == "" {
		ctx = ctx.WithChainID(eth.EIP155ChainID_Testnet)
	}
	return next(ctx, tx, simulate)
}
