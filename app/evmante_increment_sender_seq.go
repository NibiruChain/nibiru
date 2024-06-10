// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/x/evm"
)

var (
	_ sdk.AnteDecorator = (*AnteDecEthGasConsume)(nil)
	_ sdk.AnteDecorator = (*AnteDecVerifyEthAcc)(nil)
)

// AnteDecEthIncrementSenderSequence increments the sequence of the signers.
type AnteDecEthIncrementSenderSequence struct {
	AppKeepers
}

// NewAnteDecEthIncrementSenderSequence creates a new EthIncrementSenderSequenceDecorator.
func NewAnteDecEthIncrementSenderSequence(k AppKeepers) AnteDecEthIncrementSenderSequence {
	return AnteDecEthIncrementSenderSequence{
		AppKeepers: k,
	}
}

// AnteHandle handles incrementing the sequence of the signer (i.e. sender). If the transaction is a
// contract creation, the nonce will be incremented during the transaction execution and not within
// this AnteHandler decorator.
func (issd AnteDecEthIncrementSenderSequence) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil))
		}

		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errors.Wrap(err, "failed to unpack tx data")
		}

		// increase sequence of sender
		acc := issd.AccountKeeper.GetAccount(ctx, msgEthTx.GetFrom())
		if acc == nil {
			return ctx, errors.Wrapf(
				errortypes.ErrUnknownAddress,
				"account %s is nil", gethcommon.BytesToAddress(msgEthTx.GetFrom().Bytes()),
			)
		}
		nonce := acc.GetSequence()

		// we merged the nonce verification to nonce increment, so when tx includes multiple messages
		// with same sender, they'll be accepted.
		if txData.GetNonce() != nonce {
			return ctx, errors.Wrapf(
				errortypes.ErrInvalidSequence,
				"invalid nonce; got %d, expected %d", txData.GetNonce(), nonce,
			)
		}

		if err := acc.SetSequence(nonce + 1); err != nil {
			return ctx, errors.Wrapf(err, "failed to set sequence to %d", acc.GetSequence()+1)
		}

		issd.AccountKeeper.SetAccount(ctx, acc)
	}

	return next(ctx, tx, simulate)
}
