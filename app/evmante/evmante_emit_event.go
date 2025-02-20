// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// EthEmitEventDecorator emit events in ante handler in case of tx execution failed (out of block gas limit).
type EthEmitEventDecorator struct {
	evmKeeper *EVMKeeper
}

// NewEthEmitEventDecorator creates a new EthEmitEventDecorator
func NewEthEmitEventDecorator(k *EVMKeeper) EthEmitEventDecorator {
	return EthEmitEventDecorator{
		evmKeeper: k,
	}
}

// AnteHandle emits some basic events for the eth messages
func (eeed EthEmitEventDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// After eth tx passed ante handler, the fee is deducted and nonce increased,
	// it shouldn't be ignored by json-rpc. We need to emit some events at the
	// very end of ante handler to be indexed by the consensus engine.
	txIndex := eeed.evmKeeper.EVMState().BlockTxIndex.GetOr(ctx, 0)

	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"invalid message type %T, expected %T",
				msg, (*evm.MsgEthereumTx)(nil),
			)
		}
		// Untyped event "pending_ethereum_tx" is emitted for then indexing purposes.
		// Tendermint tx_search can only search the untyped events.
		// TxHash and TxIndex values are exposed in the ante handler (before the actual tx execution)
		// to allow searching for txs which are failed due to "out of block gas limit" error.
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				evm.PendingEthereumTxEvent,
				sdk.NewAttribute(evm.PendingEthereumTxEventAttrEthHash, msgEthTx.Hash),
				sdk.NewAttribute(evm.PendingEthereumTxEventAttrIndex, strconv.FormatUint(txIndex+uint64(i), 10)),
			),
		)
	}
	return next(ctx, tx, simulate)
}
