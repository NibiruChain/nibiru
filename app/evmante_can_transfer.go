// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"math/big"

	"cosmossdk.io/errors"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
	"github.com/NibiruChain/nibiru/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
)

// CanTransferDecorator checks if the sender is allowed to transfer funds according to the EVM block
// context rules.
type CanTransferDecorator struct {
	AppKeepers
}

// NewCanTransferDecorator creates a new CanTransferDecorator instance.
func NewCanTransferDecorator(k AppKeepers) CanTransferDecorator {
	return CanTransferDecorator{
		AppKeepers: k,
	}
}

// AnteHandle creates an EVM from the message and calls the BlockContext CanTransfer function to
// see if the address can execute the transaction.
func (ctd CanTransferDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	params := ctd.EvmKeeper.GetParams(ctx)
	ethCfg := types.EthereumConfig(ctd.EvmKeeper.EthChainID(ctx))
	signer := gethcore.MakeSigner(ethCfg, big.NewInt(ctx.BlockHeight()))

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*types.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(
				errortypes.ErrUnknownRequest,
				"invalid message type %T, expected %T", msg, (*types.MsgEthereumTx)(nil),
			)
		}
		baseFee := ctd.EvmKeeper.GetBaseFee(ctx)

		coreMsg, err := msgEthTx.AsMessage(signer, baseFee)
		if err != nil {
			return ctx, errors.Wrapf(
				err,
				"failed to create an ethereum core.Message from signer %T", signer,
			)
		}

		if baseFee == nil {
			return ctx, errors.Wrap(
				types.ErrInvalidBaseFee,
				"base fee is supported but evm block context value is nil",
			)
		}
		if coreMsg.GasFeeCap().Cmp(baseFee) < 0 {
			return ctx, errors.Wrapf(
				errortypes.ErrInsufficientFee,
				"max fee per gas less than block base fee (%s < %s)",
				coreMsg.GasFeeCap(), baseFee,
			)
		}

		// NOTE: pass in an empty coinbase address and nil tracer as we don't need them for the check below
		cfg := &statedb.EVMConfig{
			ChainConfig: ethCfg,
			Params:      params,
			CoinBase:    gethcommon.Address{},
			BaseFee:     baseFee,
		}

		stateDB := statedb.New(
			ctx,
			&ctd.EvmKeeper,
			statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes())),
		)
		evmInstance := ctd.EvmKeeper.NewEVM(ctx, coreMsg, cfg, types.NewNoOpTracer(), stateDB)

		// check that caller has enough balance to cover asset transfer for **topmost** call
		// NOTE: here the gas consumed is from the context with the infinite gas meter
		if coreMsg.Value().Sign() > 0 &&
			!evmInstance.Context.CanTransfer(stateDB, coreMsg.From(), coreMsg.Value()) {
			return ctx, errors.Wrapf(
				errortypes.ErrInsufficientFunds,
				"failed to transfer %s from address %s using the EVM block context transfer function",
				coreMsg.Value(),
				coreMsg.From(),
			)
		}
	}

	return next(ctx, tx, simulate)
}
