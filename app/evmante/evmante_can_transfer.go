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

		baseFeeWeiPerGas := evm.NativeToWei(ctd.EVMKeeper.BaseFeeMicronibiPerGas(ctx))

		coreMsg, err := msgEthTx.AsMessage(signer, baseFeeWeiPerGas)
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
		if coreMsg.GasFeeCap().Cmp(baseFeeWeiPerGas) < 0 {
			return ctx, errors.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"gas fee cap (wei) less than block base fee (wei); (%s < %s)",
				coreMsg.GasFeeCap(), baseFeeWeiPerGas,
			)
		}

		cfg := &statedb.EVMConfig{
			ChainConfig: ethCfg,
			Params:      params,
			// Note that we use an empty coinbase here  because the field is not
			// used during this Ante Handler.
			BlockCoinbase: gethcommon.Address{},
			BaseFeeWei:    baseFeeWeiPerGas,
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
