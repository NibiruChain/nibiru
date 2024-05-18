// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app/ante"
)

// NewAnteHandlerEVM creates the default ante handler for Ethereum transactions
func NewAnteHandlerEVM(
	k AppKeepers, options ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// outermost AnteDecorator. SetUpContext must be called first
		NewEthSetUpContextDecorator(k),
		// Check eth effective gas price against the node's minimal-gas-prices config
		NewEthMempoolFeeDecorator(k),
		// Check eth effective gas price against the global MinGasPrice
		NewEthMinGasPriceDecorator(k),
		NewEthValidateBasicDecorator(k),
		NewEthSigVerificationDecorator(k),
		NewEthAccountVerificationDecorator(k),
		NewCanTransferDecorator(k),
		NewEthGasConsumeDecorator(k, options.MaxTxGasWanted),
		NewEthIncrementSenderSequenceDecorator(k),
		NewGasWantedDecorator(k),
		// emit eth tx hash and index at the very last ante handler.
		NewEthEmitEventDecorator(k),
	)
}
