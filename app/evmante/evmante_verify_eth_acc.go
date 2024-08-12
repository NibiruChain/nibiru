// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

// AnteDecVerifyEthAcc validates an account balance checks
type AnteDecVerifyEthAcc struct {
	evmKeeper     EVMKeeper
	accountKeeper evm.AccountKeeper
}

// NewAnteDecVerifyEthAcc creates a new EthAccountVerificationDecorator
func NewAnteDecVerifyEthAcc(k EVMKeeper, ak evm.AccountKeeper) AnteDecVerifyEthAcc {
	return AnteDecVerifyEthAcc{
		evmKeeper:     k,
		accountKeeper: ak,
	}
}

// AnteHandle validates checks that the sender balance is greater than the total transaction cost.
// The account will be set to store if it doesn't exist, i.e. cannot be found on store.
// This AnteHandler decorator will fail if:
// - any of the msgs is not a MsgEthereumTx
// - from address is empty
// - account balance is lower than the transaction cost
func (anteDec AnteDecVerifyEthAcc) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil))
		}

		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errors.Wrapf(err, "failed to unpack tx data any for tx %d", i)
		}

		// sender address should be in the tx cache from the previous AnteHandle call
		from := msgEthTx.GetFrom()
		if from.Empty() {
			return ctx, errors.Wrap(errortypes.ErrInvalidAddress, "from address cannot be empty")
		}

		// check whether the sender address is EOA
		fromAddr := gethcommon.BytesToAddress(from)
		acct := anteDec.evmKeeper.GetAccount(ctx, fromAddr)

		if acct == nil {
			acc := anteDec.accountKeeper.NewAccountWithAddress(ctx, from)
			anteDec.accountKeeper.SetAccount(ctx, acc)
			acct = statedb.NewEmptyAccount()
		} else if acct.IsContract() {
			return ctx, errors.Wrapf(errortypes.ErrInvalidType,
				"the sender is not EOA: address %s, codeHash <%s>", fromAddr, acct.CodeHash)
		}

		if err := checkSenderBalance(
			evm.NativeToWei(acct.BalanceNative), txData,
		); err != nil {
			return ctx, errors.Wrap(err, "failed to check sender balance")
		}
	}
	return next(ctx, tx, simulate)
}

// checkSenderBalance validates that the tx cost value is positive and that the
// sender has enough funds to pay for the fees and value of the transaction.
func checkSenderBalance(
	balanceWei *big.Int,
	txData evm.TxData,
) error {
	cost := txData.Cost()

	if cost.Sign() < 0 {
		return errors.Wrapf(
			errortypes.ErrInvalidCoins,
			"tx cost (%s) is negative and invalid", cost,
		)
	}

	if balanceWei.Cmp(big.NewInt(0)) < 0 || balanceWei.Cmp(cost) < 0 {
		return errors.Wrapf(
			errortypes.ErrInsufficientFunds,
			"sender balance < tx cost (%s < %s)", balanceWei, cost,
		)
	}
	return nil
}
