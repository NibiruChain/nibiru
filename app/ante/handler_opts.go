package ante

import (
	sdkerrors "cosmossdk.io/errors"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	devgasante "github.com/NibiruChain/nibiru/x/devgas/v1/ante"
	devgaskeeper "github.com/NibiruChain/nibiru/x/devgas/v1/keeper"
	evmkeeper "github.com/NibiruChain/nibiru/x/evm/keeper"
)

type AnteHandlerOptions struct {
	sdkante.HandlerOptions
	IBCKeeper        *ibckeeper.Keeper
	DevGasKeeper     *devgaskeeper.Keeper
	DevGasBankKeeper devgasante.BankKeeper
	EvmKeeper        evmkeeper.Keeper
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

type TxFeeChecker func(ctx sdk.Context, feeTx sdk.FeeTx) (sdk.Coins, int64, error)
