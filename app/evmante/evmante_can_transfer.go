// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
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
	params := ctd.GetParams(ctx)
	ethCfg := evm.EthereumConfig(ctd.EVMKeeper.EthChainID(ctx))
	signer := gethcore.MakeSigner(ethCfg, big.NewInt(ctx.BlockHeight()))

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil),
			)
		}
		baseFeeMicronibiPerGas := ctd.EVMKeeper.BaseFeeMicronibiPerGas(ctx)
		baseFeeWeiPerGas := evm.NativeToWei(baseFeeMicronibiPerGas)

		coreMsg, err := msgEthTx.AsMessage(signer, baseFeeWeiPerGas)
		if err != nil {
			return ctx, errors.Wrapf(
				err,
				"failed to create an ethereum core.Message from signer %T", signer,
			)
		}

		if baseFeeMicronibiPerGas == nil {
			return ctx, errors.Wrap(
				evm.ErrInvalidBaseFee,
				"base fee is supported but evm block context value is nil",
			)
		}
		if coreMsg.GasFeeCap().Cmp(baseFeeMicronibiPerGas) < 0 {
			return ctx, errors.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"max fee per gas less than block base fee (%s < %s)",
				coreMsg.GasFeeCap(), baseFeeMicronibiPerGas,
			)
		}

		// NOTE: pass in an empty coinbase address and nil tracer as we don't need them for the check below
		cfg := &statedb.EVMConfig{
			ChainConfig:   ethCfg,
			Params:        params,
			BlockCoinbase: gethcommon.Address{},
			BaseFeeWei:    baseFeeMicronibiPerGas,
		}

		stateDB := statedb.New(
			ctx,
			ctd.EVMKeeper,
			statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes())),
		)
		evmInstance := ctd.EVMKeeper.NewEVM(ctx, coreMsg, cfg, evm.NewNoOpTracer(), stateDB)

		// check that caller has enough balance to cover asset transfer for **topmost** call
		// NOTE: here the gas consumed is from the context with the infinite gas meter
		if coreMsg.Value().Sign() > 0 &&
			!evmInstance.Context.CanTransfer(stateDB, coreMsg.From(), coreMsg.Value()) {
			balanceWei := stateDB.GetBalance(coreMsg.From())
			return ctx, errors.Wrapf(
				sdkerrors.ErrInsufficientFunds,
				"failed to transfer %s wei (balance=%s) from address %s using the EVM block context transfer function",
				coreMsg.Value(),
				balanceWei,
				coreMsg.From(),
			)
		}
	}

	return next(ctx, tx, simulate)
}
