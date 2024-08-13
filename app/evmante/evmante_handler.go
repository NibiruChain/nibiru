// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
)

// NewAnteHandlerEVM creates the default ante handler for Ethereum transactions
func NewAnteHandlerEVM(
	options ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// outermost AnteDecorator. SetUpContext must be called first
		NewEthSetUpContextDecorator(&options.EvmKeeper),
		NewMempoolGasPriceDecorator(&options.EvmKeeper),
		NewEthValidateBasicDecorator(&options.EvmKeeper),
		NewEthSigVerificationDecorator(&options.EvmKeeper),
		NewAnteDecVerifyEthAcc(&options.EvmKeeper, options.AccountKeeper),
		NewCanTransferDecorator(&options.EvmKeeper),
		NewAnteDecEthGasConsume(&options.EvmKeeper, options.MaxTxGasWanted),
		NewAnteDecEthIncrementSenderSequence(&options.EvmKeeper, options.AccountKeeper),
		ante.AnteDecoratorGasWanted{},
		// emit eth tx hash and index at the very last ante handler.
		NewEthEmitEventDecorator(&options.EvmKeeper),
	)
}
