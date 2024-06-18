// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/evm/types"
)

// EthEmitEventDecorator emit events in ante handler in case of tx execution failed (out of block gas limit).
type EthEmitEventDecorator struct {
	AppKeepers
}

// NewEthEmitEventDecorator creates a new EthEmitEventDecorator
func NewEthEmitEventDecorator(k AppKeepers) EthEmitEventDecorator {
	return EthEmitEventDecorator{AppKeepers: k}
}

// AnteHandle emits some basic events for the eth messages
func (eeed EthEmitEventDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// After eth tx passed ante handler, the fee is deducted and nonce increased,
	// it shouldn't be ignored by json-rpc. We need to emit some events at the
	// very end of ante handler to be indexed by the consensus engine.
	txIndex := eeed.EvmKeeper.EVMState().BlockTxIndex.GetOr(ctx, 0)

	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*types.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrUnknownRequest,
				"invalid message type %T, expected %T",
				msg, (*types.MsgEthereumTx)(nil),
			)
		}

		// emit ethereum tx hash as an event so that it can be indexed by
		// Tendermint for query purposes it's emitted in ante handler, so we can
		// query failed transaction (out of block gas limit).
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeEthereumTx,
				sdk.NewAttribute(types.AttributeKeyEthereumTxHash, msgEthTx.Hash),
				sdk.NewAttribute(
					types.AttributeKeyTxIndex, strconv.FormatUint(txIndex+uint64(i),
						10,
					),
				), // #nosec G701
			))
	}

	return next(ctx, tx, simulate)
}
