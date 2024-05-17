// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"

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

// NewAnteHandlerNonEVM creates the default ante handler for Cosmos transactions
func NewAnteHandlerNonEVM(
	k AppKeepers, options ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		AnteDecoratorPreventEtheruemTxMsgs{}, // reject MsgEthereumTxs
		// TODO: UD
		// cosmosante.NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
		// 	sdk.MsgTypeURL(&evm.MsgEthereumTx{}),
		// 	sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
		// ),
		authante.NewSetUpContextDecorator(),
		authante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(options.AccountKeeper),
		// TODO: UD
		// cosmosante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		authante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		// TODO: UD
		// cosmosante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.DistributionKeeper, options.FeegrantKeeper, options.StakingKeeper, options.TxFeeChecker),
		// TODO: UD
		// cosmosante.NewVestingDelegationDecorator(options.AccountKeeper, options.StakingKeeper, options.BankKeeper, options.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(options.AccountKeeper),
		authante.NewValidateSigCountDecorator(options.AccountKeeper),
		authante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		authante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		authante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		NewGasWantedDecorator(k),
	)
}
