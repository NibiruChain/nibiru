package app

import (
	corestoretypes "cosmossdk.io/core/store"
	sdkerrors "cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/NibiruChain/nibiru/app/ante"
	devgasante "github.com/NibiruChain/nibiru/x/devgas/v1/ante"
	devgaskeeper "github.com/NibiruChain/nibiru/x/devgas/v1/keeper"
)

type AnteHandlerOptions struct {
	sdkante.HandlerOptions
	IBCKeeper        *ibckeeper.Keeper
	DevGasKeeper     *devgaskeeper.Keeper
	DevGasBankKeeper devgasante.BankKeeper

	TXCounterStoreService corestoretypes.KVStoreService
	WasmConfig            *wasmtypes.WasmConfig
}

// NewAnteHandler returns and AnteHandler that checks and increments sequence
// numbers, checks signatures and account numbers, and deducts fees from the
// first signer.
func NewAnteHandler(options AnteHandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, AnteHandlerError("account keeper")
	}
	if options.BankKeeper == nil {
		return nil, AnteHandlerError("bank keeper")
	}
	if options.SignModeHandler == nil {
		return nil, AnteHandlerError("sign mode handler")
	}
	if options.SigGasConsumer == nil {
		options.SigGasConsumer = sdkante.DefaultSigVerificationGasConsumer
	}
	if options.WasmConfig == nil {
		return nil, AnteHandlerError("wasm config")
	}
	if options.DevGasKeeper == nil {
		return nil, AnteHandlerError("devgas keeper")
	}
	if options.IBCKeeper == nil {
		return nil, AnteHandlerError("ibc keeper")
	}

	anteDecorators := []sdk.AnteDecorator{
		sdkante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
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

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}

func AnteHandlerError(shortDesc string) error {
	return sdkerrors.Wrapf(errors.ErrLogic, "%s is required for AnteHandler", shortDesc)
}
