// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	cmttypes "github.com/cometbft/cometbft/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/evm/evmstate"
)

var _ AnteStep = AnteStepIncrementNonce

const (
	// MaxPendingTxsPerSender is the inclusive number of transaction nonces one
	// account may hold in a node's mempool: the committed state nonce through
	// stateNonce + MaxPendingTxsPerSender - 1 (8 values when the limit is 8).
	MaxPendingTxsPerSender uint64 = 8
	// MaxFutureNonceGap is the maximum accepted distance between the transaction
	// nonce and the committed state nonce during CheckTxType_New:
	// txNonce - stateNonce must be strictly less than [MaxPendingTxsPerSender],
	// so MaxFutureNonceGap equals MaxPendingTxsPerSender - 1.
	MaxFutureNonceGap uint64 = MaxPendingTxsPerSender - 1
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
// During CheckTxType_New, the step admits transaction nonces in the inclusive
// window from the committed state nonce through stateNonce + [MaxFutureNonceGap]
// (rejecting when txNonce - stateNonce >= [MaxPendingTxsPerSender]).
// [AnteStepMempoolAdmission] and [evm.Mempool.Insert] enforce live-slot
// ownership. During CheckTxType_Recheck, the step retains complete state nonce
// chains via [evm.Mempool.CheckRecheck]. Proposal and delivery execution require
// the exact current state nonce.
//
// The nonce is incremented only during DeliverTx in the active ante state.
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
	case sdb.IsReCheckTxOnly():
		mp := opts.GetEVMMempool()
		if mp == nil {
			if txNonce != stateNonce {
				return fmt.Errorf(
					"invalid nonce; got %d, expected %d", txNonce, stateNonce,
				)
			}
			return nil
		}
		return mp.CheckRecheck(
			cmttypes.Tx(sdb.Ctx().TxBytes()).Key(),
			msgEthTx.FromAddr(),
			stateNonce,
			txNonce,
		)
	case sdb.IsDeliverTx():
		if txNonce != stateNonce {
			return fmt.Errorf(
				"invalid nonce; got %d, expected %d", txNonce, stateNonce,
			)
		}
		sdb.SetNonce(msgEthTx.FromAddr(), stateNonce+1)
	case sdb.Ctx().IsCheckTx():
		if txNonce < stateNonce {
			return fmt.Errorf(
				"invalid nonce; got %d, expected %d or higher", txNonce, stateNonce,
			)
		}
		if txNonce-stateNonce >= MaxPendingTxsPerSender {
			return fmt.Errorf(
				"future nonce gap too large; got %d, state nonce %d, max gap %d",
				txNonce, stateNonce, MaxFutureNonceGap,
			)
		}
	default:
	}

	return nil
}
