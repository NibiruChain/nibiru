// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/eth"
)

// AnteDecoratorGasWanted keeps track of the gasWanted amount on the current block in
// transient store for BaseFee calculation.
type AnteDecoratorGasWanted struct{}

func (gwd AnteDecoratorGasWanted) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}

	gasWanted := feeTx.GetGas()
	// return error if the tx gas is greater than the block limit (max gas)
	blockGasLimit := eth.BlockGasLimit(ctx)
	if gasWanted > blockGasLimit {
		return ctx, errors.Wrapf(
			errortypes.ErrOutOfGas,
			"tx gas (%d) exceeds block gas limit (%d)",
			gasWanted,
			blockGasLimit,
		)
	}

	return next(ctx, tx, simulate)
}
