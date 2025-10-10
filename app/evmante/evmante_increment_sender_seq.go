// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

// AnteDecEthIncrementSenderSequence increments the sequence of the signers.
type AnteDecEthIncrementSenderSequence struct {
	evmKeeper     *EVMKeeper
	accountKeeper ante.AccountKeeper
}

// NewAnteDecEthIncrementSenderSequence creates a new EthIncrementSenderSequenceDecorator.
func NewAnteDecEthIncrementSenderSequence(k *EVMKeeper, ak ante.AccountKeeper) AnteDecEthIncrementSenderSequence {
	return AnteDecEthIncrementSenderSequence{
		evmKeeper:     k,
		accountKeeper: ak,
	}
}

// AnteHandle handles incrementing the sequence of the signer (i.e. sender). If the transaction is a
// contract creation, the nonce will be incremented during the transaction execution and not within
// this AnteHandler decorator.
func (issd AnteDecEthIncrementSenderSequence) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	msgEthTx, err := evm.RequireStandardEVMTxMsg(tx)
	if err != nil {
		return ctx, err
	}

	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return ctx, sdkioerrors.Wrap(err, "failed to unpack tx data")
	}

	// increase sequence of sender
	acc := issd.accountKeeper.GetAccount(ctx, msgEthTx.FromAddrBech32())
	if acc == nil {
		return ctx, sdkioerrors.Wrapf(
			sdkerrors.ErrUnknownAddress,
			"account %s is nil", gethcommon.BytesToAddress(msgEthTx.FromAddrBech32().Bytes()),
		)
	}
	nonce := acc.GetSequence()

	// we merged the nonce verification to nonce increment, so when tx includes multiple messages
	// with same sender, they'll be accepted.
	if txData.GetNonce() != nonce {
		return ctx, sdkioerrors.Wrapf(
			sdkerrors.ErrInvalidSequence,
			"invalid nonce; got %d, expected %d", txData.GetNonce(), nonce,
		)
	}

	if err := acc.SetSequence(nonce + 1); err != nil {
		return ctx, sdkioerrors.Wrapf(err, "failed to set sequence to %d", acc.GetSequence()+1)
	}

	issd.accountKeeper.SetAccount(ctx, acc)

	return next(ctx, tx, simulate)
}

var _ EvmAnteHandler = EthAnteIncrementNonce

func EthAnteIncrementNonce(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	acc := k.GetAccount(sdb.Ctx(), msgEthTx.FromAddr())
	if acc == nil {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrUnknownAddress,
			"account %s is nil", gethcommon.BytesToAddress(msgEthTx.FromAddrBech32().Bytes()),
		)
	}

	nonce := acc.Nonce

	// we merged the nonce verification to nonce increment, so when tx includes multiple messages
	// with same sender, they'll be accepted.
	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to unpack tx data")
	}
	if txData.GetNonce() != nonce {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInvalidSequence,
			"invalid nonce; got %d, expected %d", txData.GetNonce(), nonce,
		)
	}

	newNonce := nonce + 1
	sdb.SetNonce(msgEthTx.FromAddr(), newNonce)

	return nil
}
