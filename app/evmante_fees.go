// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/x/evm"
)

var (
	_ sdk.AnteDecorator = EthMinGasPriceDecorator{}
	_ sdk.AnteDecorator = EthMempoolFeeDecorator{}
)

// EthMinGasPriceDecorator will check if the transaction's fee is at least as large
// as the MinGasPrices param. If fee is too low, decorator returns error and tx
// is rejected. This applies to both CheckTx and DeliverTx and regardless
// if London hard fork or fee market params (EIP-1559) are enabled.
// If fee is high enough, then call next AnteHandler
type EthMinGasPriceDecorator struct {
	AppKeepers
}

// EthMempoolFeeDecorator will check if the transaction's effective fee is at
// least as large as the local validator's minimum gasFee (defined in validator
// config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeDecorator
type EthMempoolFeeDecorator struct {
	AppKeepers
}

// NewEthMinGasPriceDecorator creates a new MinGasPriceDecorator instance used only for
// Ethereum transactions.
func NewEthMinGasPriceDecorator(k AppKeepers) EthMinGasPriceDecorator {
	return EthMinGasPriceDecorator{AppKeepers: k}
}

// NewEthMempoolFeeDecorator creates a new NewEthMempoolFeeDecorator instance used only for
// Ethereum transactions.
func NewEthMempoolFeeDecorator(k AppKeepers) EthMempoolFeeDecorator {
	return EthMempoolFeeDecorator{
		AppKeepers: k,
	}
}

// AnteHandle ensures that the effective fee from the transaction is greater than the
// minimum global fee, which is defined by the  MinGasPrice (parameter) * GasLimit (tx argument).
func (empd EthMinGasPriceDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	minGasPrices := ctx.MinGasPrices()
	evmParams := empd.EvmKeeper.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	minGasPrice := minGasPrices.AmountOf(evmDenom)

	// short-circuit if min gas price is 0
	if minGasPrice.IsZero() {
		return next(ctx, tx, simulate)
	}

	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(empd.EvmKeeper.EthChainID(ctx))
	baseFee := empd.EvmKeeper.GetBaseFee(ctx, ethCfg)

	for _, msg := range tx.GetMsgs() {
		ethMsg, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(
				errortypes.ErrUnknownRequest,
				"invalid message type %T, expected %T",
				msg, (*evm.MsgEthereumTx)(nil),
			)
		}

		feeAmt := ethMsg.GetFee()

		// For dynamic transactions, GetFee() uses the GasFeeCap value, which
		// is the maximum gas price that the signer can pay. In practice, the
		// signer can pay less, if the block's BaseFee is lower. So, in this case,
		// we use the EffectiveFee. If the feemarket formula results in a BaseFee
		// that lowers EffectivePrice until it is < MinGasPrices, the users must
		// increase the GasTipCap (priority fee) until EffectivePrice > MinGasPrices.
		// Transactions with MinGasPrices * gasUsed < tx fees < EffectiveFee are rejected
		// by the feemarket AnteHandle

		txData, err := evm.UnpackTxData(ethMsg.Data)
		if err != nil {
			return ctx, errors.Wrapf(err, "failed to unpack tx data %s", ethMsg.Hash)
		}

		if txData.TxType() != gethcore.LegacyTxType {
			feeAmt = ethMsg.GetEffectiveFee(baseFee)
		}

		gasLimit := sdk.NewDecFromBigInt(new(big.Int).SetUint64(ethMsg.GetGas()))

		requiredFee := minGasPrice.Mul(gasLimit)
		fee := sdk.NewDecFromBigInt(feeAmt)

		if fee.LT(requiredFee) {
			return ctx, errors.Wrapf(
				errortypes.ErrInsufficientFee,
				"provided fee < minimum global fee (%s < %s). Please increase the priority tip (for EIP-1559 txs) or the gas prices (for access list or legacy txs)", //nolint:lll
				fee.TruncateInt().String(), requiredFee.TruncateInt().String(),
			)
		}
	}

	return next(ctx, tx, simulate)
}

// AnteHandle ensures that the provided fees meet a minimum threshold for the validator.
// This check only for local mempool purposes, and thus it is only run on (Re)CheckTx.
// The logic is also skipped if the London hard fork and EIP-1559 are enabled.
func (mfd EthMempoolFeeDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	evmParams := mfd.EvmKeeper.GetParams(ctx)
	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(mfd.EvmKeeper.EthChainID(ctx))

	baseFee := mfd.EvmKeeper.GetBaseFee(ctx, ethCfg)
	// skip check as the London hard fork and EIP-1559 are enabled
	if baseFee != nil {
		return next(ctx, tx, simulate)
	}

	evmDenom := evmParams.GetEvmDenom()
	minGasPrice := ctx.MinGasPrices().AmountOf(evmDenom)

	for _, msg := range tx.GetMsgs() {
		ethMsg, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil))
		}

		fee := sdk.NewDecFromBigInt(ethMsg.GetFee())
		gasLimit := sdk.NewDecFromBigInt(new(big.Int).SetUint64(ethMsg.GetGas()))
		requiredFee := minGasPrice.Mul(gasLimit)

		if fee.LT(requiredFee) {
			return ctx, errors.Wrapf(
				errortypes.ErrInsufficientFee,
				"insufficient fee; got: %s required: %s",
				fee, requiredFee,
			)
		}
	}

	return next(ctx, tx, simulate)
}
