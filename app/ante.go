package app

import (
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"

	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/evmante"
	devgasante "github.com/NibiruChain/nibiru/v2/x/devgas/v1/ante"
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
		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 0 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/eth.evm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler = evmante.NewAnteHandlerEVM(options)
				default:
					return ctx, fmt.Errorf(
						"rejecting tx with unsupported extension option: %s", typeURL)
				}

				return anteHandler(ctx, tx, sim)
			}
		}

		switch tx.(type) {
		case sdk.Tx:
			anteHandler = NewAnteHandlerNonEVM(options)
		default:
			return ctx, fmt.Errorf("invalid tx type (%T) in AnteHandler", tx)
		}
		return anteHandler(ctx, tx, sim)
	}
}

// NewAnteHandlerNonEVM: Default ante handler for non-EVM transactions.
func NewAnteHandlerNonEVM(
	opts ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ante.AnteDecoratorPreventEtheruemTxMsgs{}, // reject MsgEthereumTxs
		ante.AnteDecoratorAuthzGuard{},            // disable certain messages in authz grant "generic"
		authante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(opts.TxCounterStoreKey),
		// TODO: bug(security): Authz is unsafe. Let's include a guard to make
		// things safer.
		// ticket: https://github.com/NibiruChain/nibiru/issues/1915
		authante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.AnteDecoratorEnsureSinglePostPriceMessage{},
		ante.AnteDecoratorStakingCommission{},
		// ----------- Ante Handlers: Gas
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		// TODO: spike(security): Does minimum gas price of 0 pose a risk?
		// ticket: https://github.com/NibiruChain/nibiru/issues/1916
		authante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		// ----------- Ante Handlers:  devgas
		devgasante.NewDevGasPayoutDecorator(opts.DevGasBankKeeper, opts.DevGasKeeper),
		// ----------- Ante Handlers:  Keys and signatures
		// NOTE: SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(opts.AccountKeeper),
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, opts.SigGasConsumer),
		authante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
		ante.AnteDecoratorGasWanted{},
	)
}
