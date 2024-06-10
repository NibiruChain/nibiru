// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
	"github.com/NibiruChain/nibiru/x/evm/statedb"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	_ sdk.AnteDecorator = (*AnteDecEthGasConsume)(nil)
	_ sdk.AnteDecorator = (*AnteDecVerifyEthAcc)(nil)
)

// AnteDecVerifyEthAcc validates an account balance checks
type AnteDecVerifyEthAcc struct {
	AppKeepers
}

// NewAnteDecVerifyEthAcc creates a new EthAccountVerificationDecorator
func NewAnteDecVerifyEthAcc(k AppKeepers) AnteDecVerifyEthAcc {
	return AnteDecVerifyEthAcc{
		AppKeepers: k,
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
	if !ctx.IsCheckTx() {
		return next(ctx, tx, simulate)
	}

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
		acct := anteDec.EvmKeeper.GetAccount(ctx, fromAddr)

		if acct == nil {
			acc := anteDec.AccountKeeper.NewAccountWithAddress(ctx, from)
			anteDec.AccountKeeper.SetAccount(ctx, acc)
			acct = statedb.NewEmptyAccount()
		} else if acct.IsContract() {
			return ctx, errors.Wrapf(errortypes.ErrInvalidType,
				"the sender is not EOA: address %s, codeHash <%s>", fromAddr, acct.CodeHash)
		}

		if err := keeper.CheckSenderBalance(sdkmath.NewIntFromBigInt(acct.Balance), txData); err != nil {
			return ctx, errors.Wrap(err, "failed to check sender balance")
		}
	}
	return next(ctx, tx, simulate)
}
