// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ AnteStep = AnteStepIncrementNonce

// AnteStepIncrementNonce increments the sequence (nonce) of the sender account
// and validates that the transaction nonce matches the expected account nonce.
// This handler manages nonce verification and increment for Ethereum transactions.
//
// This handler will fail if:
//   - the account does not exist
//   - the transaction nonce is invalid for the current context
//   - transaction data cannot be unpacked
//
// During CheckTx: Allows nonce >= current nonce (for pending transactions)
// During ReCheckTx/DeliverTx: Requires exact nonce match
//
// The nonce is incremented in the SDB state, ensuring proper sequencing
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
				"invalid nonce; got %d, should be expected %d or higher with pending txs in the same block", txNonce, stateNonce,
			)
		}
	case sdb.Ctx().IsReCheckTx() || sdb.IsDeliverTx():
		if txNonce != stateNonce {
			return fmt.Errorf(
				"invalid nonce; got %d, expected %d", txNonce, stateNonce,
			)
		}
		newNonce := stateNonce + 1
		sdb.SetNonce(msgEthTx.FromAddr(), newNonce)
	default:
	}

	return nil
}
