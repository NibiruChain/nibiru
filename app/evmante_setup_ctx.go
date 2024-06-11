// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// EthSetupContextDecorator is adapted from SetUpContextDecorator from cosmos-sdk, it ignores gas consumption
// by setting the gas meter to infinite
type EthSetupContextDecorator struct {
	AppKeepers
}

func NewEthSetUpContextDecorator(k AppKeepers) EthSetupContextDecorator {
	return EthSetupContextDecorator{
		AppKeepers: k,
	}
}

func (esc EthSetupContextDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// all transactions must implement GasTx
	_, ok := tx.(authante.GasTx)
	if !ok {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrInvalidType,
			"invalid transaction type %T, expected GasTx", tx,
		)
	}

	// We need to setup an empty gas config so that the gas is consistent with Ethereum.
	newCtx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter()).
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})

	// Reset transient gas used to prepare the execution of current cosmos tx.
	// Transient gas-used is necessary to sum the gas-used of cosmos tx, when it contains multiple eth msgs.
	esc.EvmKeeper.ResetTransientGasUsed(ctx)
	return next(newCtx, tx, simulate)
}
