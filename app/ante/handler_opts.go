package ante

import (
	"cosmossdk.io/core/store"
	sdkerrors "cosmossdk.io/errors"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	devgasante "github.com/NibiruChain/nibiru/x/devgas/v1/ante"
	devgaskeeper "github.com/NibiruChain/nibiru/x/devgas/v1/keeper"
)

type AnteHandlerOptions struct {
	sdkante.HandlerOptions
	IBCKeeper        *ibckeeper.Keeper
	DevGasKeeper     *devgaskeeper.Keeper
	DevGasBankKeeper devgasante.BankKeeper

	TxCounterStoreKey store.KVStoreService
	WasmConfig        *wasmtypes.WasmConfig
	MaxTxGasWanted    uint64
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

type TxFeeChecker func(ctx sdk.Context, feeTx sdk.FeeTx) (sdk.Coins, int64, error)
