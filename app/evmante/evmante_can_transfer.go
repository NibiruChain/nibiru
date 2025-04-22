// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// CanTransferDecorator checks if the sender is allowed to transfer funds according to the EVM block
// context rules.
type CanTransferDecorator struct {
	*EVMKeeper
}

// AnteHandle creates an EVM from the message and calls the BlockContext CanTransfer function to
// see if the address can execute the transaction.
func (ctd CanTransferDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	ethCfg := evm.EthereumConfig(ctd.EVMKeeper.EthChainID(ctx))
	signer := gethcore.MakeSigner(
		ethCfg,
		big.NewInt(ctx.BlockHeight()),
		evm.ParseBlockTimeUnixU64(ctx),
	)

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil),
			)
		}

		baseFeeWeiPerGas := evm.NativeToWei(ctd.EVMKeeper.BaseFeeMicronibiPerGas(ctx))

		evmMsg, err := msgEthTx.AsMessage(signer, baseFeeWeiPerGas)
		if err != nil {
			return ctx, errors.Wrapf(
				err,
				"failed to create an ethereum core.Message from signer %T", signer,
			)
		}

		if baseFeeWeiPerGas == nil {
			return ctx, errors.Wrap(
				evm.ErrInvalidBaseFee,
				"base fee is nil for this block.",
			)
		}

		if msgEthTx.EffectiveGasCapWei(baseFeeWeiPerGas).Cmp(baseFeeWeiPerGas) < 0 {
			return ctx, errors.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"gas fee cap (wei) less than block base fee (wei); (%s < %s)",
				evmMsg.GasFeeCap, baseFeeWeiPerGas,
			)
		}

		// check that caller has enough balance to cover asset transfer for **topmost** call
		// NOTE: here the gas consumed is from the context with the infinite gas meter

		if evmMsg.Value.Sign() > 0 {
			nibiruAddr := eth.EthAddrToNibiruAddr(evmMsg.From)
			balanceNative := ctd.Bank.GetBalance(ctx, nibiruAddr, evm.EVMBankDenom).Amount.BigInt()
			balanceWei := evm.NativeToWei(balanceNative)

			if balanceWei.Cmp(evmMsg.Value) < 0 {
				return ctx, errors.Wrapf(
					sdkerrors.ErrInsufficientFunds,
					"failed to transfer %s wei ( balance=%s )from address %s using the EVM block context transfer function",
					evmMsg.Value,
					balanceWei,
					evmMsg.From,
				)
			}
		}
	}

	return next(ctx, tx, simulate)
}
