// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/evm/evmstate"
)

var _ AnteStep = AnteStepIncrementNonce

const (
	// MaxPendingTxsPerSender bounds how many transactions one EVM account can
	// place in a node's mempool between commits.
	MaxPendingTxsPerSender uint64 = 64
	// MaxFutureNonceGap bounds the node-local queue of out-of-order transactions
	// while retaining normal future-nonce and replacement admission.
	MaxFutureNonceGap uint64 = 64
)

// AnteStepIncrementNonce increments the sequence (nonce) of the sender account
// and validates that the transaction nonce matches the expected account nonce.
// This handler manages nonce verification and increment for Ethereum transactions.
//
// This handler will fail if:
//   - the account does not exist
//   - the transaction nonce is invalid for the current context
//   - transaction data cannot be unpacked
//
// During CheckTx/ReCheckTx: Allows a bounded future-nonce window and caps the
// number of pending transactions per sender. During DeliverTx: Requires an exact
// nonce match.
//
// The nonce is incremented in the active ante state, ensuring proper sequencing
// of transactions from the same sender.
func AnteStepIncrementNonce(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	var (
		txNonce    uint64 // Nonce specified by the tx payload.
		stateNonce uint64 // Nonce of the account in the current execution ctx
	)

	acc := k.GetAccount(sdb.Ctx(), msgEthTx.FromAddr())
	if acc == nil {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrUnknownAddress,
			"account %s is nil", gethcommon.BytesToAddress(msgEthTx.FromAddrBech32().Bytes()),
		)
	}
	stateNonce = acc.Nonce

	// we merged the nonce verification to nonce increment, so when tx includes multiple messages
	// with same sender, they'll be accepted.
	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to unpack tx data")
	}
	txNonce = txData.GetNonce()

	switch {
	case sdb.Ctx().IsCheckTx():
		if txNonce < stateNonce {
			return fmt.Errorf(
				"invalid nonce; got %d, expected %d or higher", txNonce, stateNonce,
			)
		}
		if txNonce-stateNonce > MaxFutureNonceGap {
			return fmt.Errorf(
				"future nonce gap too large; got %d, state nonce %d, max gap %d",
				txNonce, stateNonce, MaxFutureNonceGap,
			)
		}

		pendingTxCount := k.EVMState().PendingTxCount.GetOr(
			sdb.RootCtx(), msgEthTx.FromAddr(), 0,
		)
		if pendingTxCount >= MaxPendingTxsPerSender {
			return fmt.Errorf(
				"too many pending transactions for sender; got %d, limit %d",
				pendingTxCount, MaxPendingTxsPerSender,
			)
		}
		k.EVMState().PendingTxCount.Insert(
			sdb.RootCtx(), msgEthTx.FromAddr(), pendingTxCount+1,
		)
	case sdb.IsDeliverTx():
		if txNonce != stateNonce {
			return fmt.Errorf(
				"invalid nonce; got %d, expected %d", txNonce, stateNonce,
			)
		}
		sdb.SetNonce(msgEthTx.FromAddr(), stateNonce+1)
	default:
	}

	return nil
}
