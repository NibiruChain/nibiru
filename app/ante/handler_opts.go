package ante

import (
	sdkioerrors "cosmossdk.io/errors"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
	sdkante "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/keeper"

	ibckeeper "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/keeper"

	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"

	"github.com/NibiruChain/nibiru/v2/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/evm/evmstate"
	devgasante "github.com/NibiruChain/nibiru/v2/x/devgas/v1/ante"
	devgaskeeper "github.com/NibiruChain/nibiru/v2/x/devgas/v1/keeper"
)

type AnteHandlerOptions struct {
	sdkante.HandlerOptions
	IBCKeeper        *ibckeeper.Keeper
	DevGasKeeper     *devgaskeeper.Keeper
	DevGasBankKeeper devgasante.BankKeeper
	EvmKeeper        *evmstate.Keeper
	EvmMempool       *evm.Mempool
	AccountKeeper    authkeeper.AccountKeeper

	TxCounterStoreKey types.StoreKey
	WasmConfig        *wasmtypes.WasmConfig
	MaxTxGasWanted    uint64
}

func (opts *AnteHandlerOptions) ValidateAndClean() error {
	if opts.BankKeeper == nil {
		return AnteHandlerError("bank keeper")
	}
	if opts.SignModeHandler == nil {
		return AnteHandlerError("sign mode handler")
	}
	opts.SigGasConsumer = NibiruSigVerificationGasConsumer
	if opts.WasmConfig == nil {
		return AnteHandlerError("wasm config")
	}
	if opts.DevGasKeeper == nil {
		return AnteHandlerError("devgas keeper")
	}
	if opts.IBCKeeper == nil {
		return AnteHandlerError("ibc keeper")
	}
	if opts.EvmMempool == nil {
		return AnteHandlerError("evm mempool")
	}
	return nil
}

func AnteHandlerError(shortDesc string) error {
	return sdkioerrors.Wrapf(sdkerrors.ErrLogic, "%s is required for AnteHandler", shortDesc)
}

// Implements the evmante.AnteOptionsEVM interface.
func (opts AnteHandlerOptions) GetMaxTxGasWanted() uint64 {
	return opts.MaxTxGasWanted
}

// GetEVMMempool returns the node-local [evm.Mempool] used by EVM ante steps such
// as [evmante.AnteStepMempoolAdmission] and [evmante.AnteStepIncrementNonce].
func (opts AnteHandlerOptions) GetEVMMempool() *evm.Mempool {
	return opts.EvmMempool
}
