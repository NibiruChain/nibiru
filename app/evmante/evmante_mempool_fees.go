// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var _ sdk.AnteDecorator = MempoolGasPriceDecorator{}

// MempoolGasPriceDecorator will check if the transaction's fee is at least as large
// as the mempool MinGasPrices param. If fee is too low, decorator returns error and tx
// is rejected. This applies to CheckTx only.
// If fee is high enough, then call next AnteHandler
type MempoolGasPriceDecorator struct {
	evmKeeper EVMKeeper
}

// NewMempoolGasPriceDecorator creates a new MinGasPriceDecorator instance used only for
// Ethereum transactions.
func NewMempoolGasPriceDecorator(k EVMKeeper) MempoolGasPriceDecorator {
	return MempoolGasPriceDecorator{
		evmKeeper: k,
	}
}

// AnteHandle ensures that the effective fee from the transaction is greater than the
// local mempool gas prices, which is defined by the  MinGasPrice (parameter) * GasLimit (tx argument).
func (d MempoolGasPriceDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// only run on CheckTx
	if !ctx.IsCheckTx() && !simulate {
		return next(ctx, tx, simulate)
	}

	minGasPrice := ctx.MinGasPrices().AmountOf(d.evmKeeper.GetParams(ctx).EvmDenom)
	// if MinGasPrices is not set, skip the check
	if minGasPrice.IsZero() {
		return next(ctx, tx, simulate)
	}

	baseFeeMicronibi := d.evmKeeper.GetBaseFee(ctx)
	baseFeeWei := evm.NativeToWei(baseFeeMicronibi)

	for _, msg := range tx.GetMsgs() {
		ethTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"invalid message type %T, expected %T",
				msg, (*evm.MsgEthereumTx)(nil),
			)
		}

		effectiveGasPriceWei := ethTx.GetEffectiveGasPrice(baseFeeWei)
		effectiveGasPrice := evm.WeiToNative(effectiveGasPriceWei)

		if sdk.NewDecFromBigInt(effectiveGasPrice).LT(minGasPrice) {
			return ctx, errors.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"provided gas price < minimum local gas price (%s < %s). "+
					"Please increase the priority tip (for EIP-1559 txs) or the gas prices "+
					"(for access list or legacy txs)",
				effectiveGasPrice.String(), minGasPrice.String(),
			)
		}
	}

	return next(ctx, tx, simulate)
}
