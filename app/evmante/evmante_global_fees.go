// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/evm"
)

var _ sdk.AnteDecorator = GlobalGasPriceDecorator{}

// GlobalGasPriceDecorator will check if the transaction's fee is at least as large
// as the global MinGasPrices param. If fee is too low, decorator returns error and tx
// is rejected. This applies to both CheckTx and DeliverTx.
// If fee is high enough, then call next AnteHandler
type GlobalGasPriceDecorator struct {
	evmKeeper EVMKeeper
}

// NewGlobalGasPriceDecorator creates a new GlobalGasPriceDecorator instance used only for
// Ethereum transactions.
func NewGlobalGasPriceDecorator(k EVMKeeper) GlobalGasPriceDecorator {
	return GlobalGasPriceDecorator{
		evmKeeper: k,
	}
}

// AnteHandle ensures that the effective fee from the transaction is greater than the
// local mempool gas prices, which is defined by the  MinGasPrice (parameter) * GasLimit (tx argument).
func (d GlobalGasPriceDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	evmParams := d.evmKeeper.GetParams(ctx)
	// if GlobalMinGasPrice is not set, skip the check
	if evmParams.GlobalMinGasPrice.IsZero() {
		return next(ctx, tx, simulate)
	}

	baseFee := d.evmKeeper.GetBaseFee(ctx)

	for _, msg := range tx.GetMsgs() {
		ethTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(
				errortypes.ErrUnknownRequest,
				"invalid message type %T, expected %T",
				msg, (*evm.MsgEthereumTx)(nil),
			)
		}

		effectiveGasPrice := ethTx.GetEffectiveGasPrice(baseFee)

		if sdk.NewDecFromBigInt(effectiveGasPrice).LT(evmParams.GlobalMinGasPrice) {
			return ctx, errors.Wrapf(
				errortypes.ErrInsufficientFee,
				"provided gas price < global gas price (%s < %s). "+
					"Please increase the priority tip (for EIP-1559 txs) or the gas prices "+
					"(for access list or legacy txs)",
				effectiveGasPrice.String(), evmParams.GlobalMinGasPrice.String(),
			)
		}
	}

	return next(ctx, tx, simulate)
}
