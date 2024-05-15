package app

import (
	sdkerrors "cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	"github.com/NibiruChain/nibiru/app/ante"
	devgasante "github.com/NibiruChain/nibiru/x/devgas/v1/ante"
	devgaskeeper "github.com/NibiruChain/nibiru/x/devgas/v1/keeper"
)

type AnteHandlerOptions struct {
	sdkante.HandlerOptions
	IBCKeeper        *ibckeeper.Keeper
	DevGasKeeper     *devgaskeeper.Keeper
	DevGasBankKeeper devgasante.BankKeeper

	TxCounterStoreKey types.StoreKey
	WasmConfig        *wasmtypes.WasmConfig
}

// NewAnteHandler returns and AnteHandler that checks and increments sequence
// numbers, checks signatures and account numbers, and deducts fees from the
// first signer.
func NewAnteHandler(options AnteHandlerOptions) (sdk.AnteHandler, error) {
	if err := options.ValidateAndClean(); err != nil {
		return nil, err
	}

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

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}

func (opts *AnteHandlerOptions) ValidateAndClean() error {
	if opts.AccountKeeper == nil {
		return AnteHandlerError("account keeper")
	}
	if opts.BankKeeper == nil {
		return AnteHandlerError("bank keeper")
	}
	if opts.SignModeHandler == nil {
		return AnteHandlerError("sign mode handler")
	}
	if opts.SigGasConsumer == nil {
		opts.SigGasConsumer = sdkante.DefaultSigVerificationGasConsumer
	}
	if opts.WasmConfig == nil {
		return AnteHandlerError("wasm config")
	}
	if opts.DevGasKeeper == nil {
		return AnteHandlerError("devgas keeper")
	}
	if opts.IBCKeeper == nil {
		return AnteHandlerError("ibc keeper")
	}
	return nil
}

func AnteHandlerError(shortDesc string) error {
	return sdkerrors.Wrapf(errors.ErrLogic, "%s is required for AnteHandler", shortDesc)
}
