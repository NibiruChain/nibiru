package app

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"

	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	devgasante "github.com/NibiruChain/nibiru/v2/x/devgas/v1/ante"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmante"
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
		if !evm.IsEthTx(tx) {
			anteHandler = NewAnteHandlerNonEVM(keepers.PublicKeepers, options)
			return anteHandler(ctx, tx, sim)
		}
		anteHandler = evmante.NewAnteHandlerEvm(options)
		return anteHandler(ctx, tx, sim)
	}
}

// NewAnteHandlerNonEVM: Default ante handler for non-EVM transactions.
func NewAnteHandlerNonEVM(
	pk keepers.PublicKeepers,
	opts ante.AnteHandlerOptions,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ante.AnteDecPreventEthereumTxMsgs{}, // reject MsgEthereumTxs
		ante.AnteDecAuthzGuard{},            // disable certain messages in authz grant "generic"
		authante.NewSetUpContextDecorator(),
		ante.AnteDecZeroGasActors{PublicKeepers: pk},
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(opts.TxCounterStoreKey),
		// TODO: bug(security): Authz is unsafe. Let's include a guard to make
		// things safer.
		// ticket: https://github.com/NibiruChain/nibiru/issues/1915
		authante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.AnteDecEnsureSinglePostPriceMessage{},
		ante.AnteDecoratorStakingCommission{},
		// ----------- Ante Handlers: Gas
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		// TODO: spike(security): Does minimum gas price of 0 pose a risk?
		// ticket: https://github.com/NibiruChain/nibiru/issues/1916
		ante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
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
		ante.AnteDecBlockGasWanted{},
	)
}
