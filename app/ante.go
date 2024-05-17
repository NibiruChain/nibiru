package app

import (
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
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
	options ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		if err := options.ValidateAndClean(); err != nil {
			return ctx, err
		}

		var anteHandler sdk.AnteHandler
		hasExt, typeUrl := TxHasExtensions(tx)
		// TODO: handle ethereum txs
		if hasExt && typeUrl != "" {
			anteHandler = AnteHandlerExtendedTx(typeUrl, keepers, options, ctx)
			return anteHandler(ctx, tx, sim)
		}

		switch tx.(type) {
		case sdk.Tx:
			anteHandler = AnteHandlerStandardTx(options)
		default:
			return ctx, fmt.Errorf("invalid tx type (%T) in AnteHandler", tx)
		}
		return anteHandler(ctx, tx, sim)
	}
}

func AnteHandlerStandardTx(options ante.AnteHandlerOptions) sdk.AnteHandler {
	anteDecorators := []sdk.AnteDecorator{
		sdkante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(options.TxCounterStoreKey),
		sdkante.NewExtensionOptionsDecorator(nil),
		sdkante.NewValidateBasicDecorator(),
		sdkante.NewTxTimeoutHeightDecorator(),
		sdkante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewPostPriceFixedPriceDecorator(),
		ante.AnteDecoratorStakingCommission{},
		sdkante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		// Replace fee ante from cosmos auth with a custom one.
		sdkante.NewDeductFeeDecorator(
			options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		devgasante.NewDevGasPayoutDecorator(
			options.DevGasBankKeeper, options.DevGasKeeper),
		// SetPubKeyDecorator must be called before all signature verification decorators
		sdkante.NewSetPubKeyDecorator(options.AccountKeeper),
		sdkante.NewValidateSigCountDecorator(options.AccountKeeper),
		sdkante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		sdkante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		sdkante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...)
}

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
