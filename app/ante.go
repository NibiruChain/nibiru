package app

import (
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"

	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/NibiruChain/nibiru/app/ante"
	"github.com/NibiruChain/nibiru/eth"
	devgasante "github.com/NibiruChain/nibiru/x/devgas/v1/ante"
	"github.com/NibiruChain/nibiru/x/evm"
)

// NewAnteHandler returns and AnteHandler that checks and increments sequence
// numbers, checks signatures and account numbers, and deducts fees from the
// first signer.
func NewAnteHandler(
	keepers AppKeepers,
	opts ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		if err := opts.ValidateAndClean(); err != nil {
			return ctx, err
		}

		var anteHandler sdk.AnteHandler
		hasExt, typeUrl := TxHasExtensions(tx)
		// TODO: handle ethereum txs
		if hasExt && typeUrl != "" {
			anteHandler = AnteHandlerExtendedTx(typeUrl, keepers, opts, ctx)
			return anteHandler(ctx, tx, sim)
		}

		switch tx.(type) {
		case sdk.Tx:
			anteHandler = NewAnteHandlerNonEVM(keepers, opts)
		default:
			return ctx, fmt.Errorf("invalid tx type (%T) in AnteHandler", tx)
		}
		return anteHandler(ctx, tx, sim)
	}
}

// TODO: UD: REMOVE ME
// func AnteHandlerStandardTx(opts ante.AnteHandlerOptions) sdk.AnteHandler {
// 	anteDecorators := []sdk.AnteDecorator{
// 		AnteDecoratorPreventEtheruemTxMsgs{}, // reject MsgEthereumTxs
// 		authante.NewSetUpContextDecorator(),
// 		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
// 		wasmkeeper.NewCountTXDecorator(opts.TxCounterStoreKey),
// 		authante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
// 		authante.NewValidateBasicDecorator(),
// 		authante.NewTxTimeoutHeightDecorator(),
// 		authante.NewValidateMemoDecorator(opts.AccountKeeper),
// 		ante.AnteDecoratorEnsureSinglePostPriceMessage{},
// 		ante.AnteDecoratorStakingCommission{},
// 		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
// 		// Replace fee ante from cosmos auth with a custom one.
// 		authante.NewDeductFeeDecorator(
// 			opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
// 		// devgas
// 		devgasante.NewDevGasPayoutDecorator(
// 			opts.DevGasBankKeeper, opts.DevGasKeeper),
// 		// NOTE: SetPubKeyDecorator must be called before all signature verification decorators
// 		authante.NewSetPubKeyDecorator(opts.AccountKeeper),
// 		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
// 		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, opts.SigGasConsumer),
// 		authante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
// 		authante.NewIncrementSequenceDecorator(opts.AccountKeeper),
// 		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
// 	}

// 	return sdk.ChainAnteDecorators(anteDecorators...)
// }

func TxHasExtensions(tx sdk.Tx) (hasExt bool, typeUrl string) {
	extensionTx, ok := tx.(authante.HasExtensionOptionsTx)
	if !ok {
		return false, ""
	}

	extOpts := extensionTx.GetExtensionOptions()
	if len(extOpts) == 0 {
		return false, ""
	}

	return true, extOpts[0].GetTypeUrl()
}

func AnteHandlerExtendedTx(
	typeUrl string,
	keepers AppKeepers,
	opts ante.AnteHandlerOptions,
	ctx sdk.Context,
) (anteHandler sdk.AnteHandler) {
	switch typeUrl {
	case evm.TYPE_URL_ETHEREUM_TX:
		anteHandler = NewAnteHandlerEVM(keepers, opts)
	case eth.TYPE_URL_DYNAMIC_FEE_TX:
		anteHandler = NewAnteHandlerNonEVM(keepers, opts)
	default:
		errUnsupported := fmt.Errorf(
			`encountered tx with unsupported extension option, "%s"`, typeUrl)
		return func(
			ctx sdk.Context, tx sdk.Tx, simulate bool,
		) (newCtx sdk.Context, err error) {
			return ctx, errUnsupported
		}
	}
	return anteHandler
}

// NewAnteHandlerNonEVM: Default ante handler for non-EVM transactions.
func NewAnteHandlerNonEVM(
	k AppKeepers, opts ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		AnteDecoratorPreventEtheruemTxMsgs{}, // reject MsgEthereumTxs
		authante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(opts.TxCounterStoreKey),
		// TODO: UD bug(security): Authz is unsafe. Let's include a guard to make
		// things safer.
		authante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.AnteDecoratorEnsureSinglePostPriceMessage{},
		ante.AnteDecoratorStakingCommission{},
		// ----------- Ante Handlers: Gas
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		// TODO: UD spike(security) Does minimum gas price of 0 pose a risk?
		authante.NewDeductFeeDecorator(
			opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		// ----------- Ante Handlers:  devgas
		devgasante.NewDevGasPayoutDecorator(
			opts.DevGasBankKeeper, opts.DevGasKeeper),
		// ----------- Ante Handlers:  Keys and signatures
		// NOTE: SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(opts.AccountKeeper),
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, opts.SigGasConsumer),
		authante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
		AnteDecoratorGasWanted{},
	)
}
