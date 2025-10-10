// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EthSetupContextDecorator is adapted from SetUpContextDecorator from cosmos-sdk, it ignores gas consumption
// by setting the gas meter to infinite
type EthSetupContextDecorator struct {
	evmKeeper *EVMKeeper
}

func NewEthSetUpContextDecorator(k *EVMKeeper) EthSetupContextDecorator {
	return EthSetupContextDecorator{
		evmKeeper: k,
	}
}

func (esc EthSetupContextDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// Set an empty gas config so that gas payment and refund is consistent with
	// Ethereum.
	newCtx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter()).
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{}).
		WithIsEvmTx(true)

	return next(newCtx, tx, simulate)
}
