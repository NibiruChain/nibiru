// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/eth"
)

// GasWantedDecorator keeps track of the gasWanted amount on the current block in transient store
// for BaseFee calculation.
// NOTE: This decorator does not perform any validation
type GasWantedDecorator struct {
	AppKeepers
}

// NewGasWantedDecorator creates a new NewGasWantedDecorator
func NewGasWantedDecorator(
	k AppKeepers,
) GasWantedDecorator {
	return GasWantedDecorator{
		AppKeepers: k,
	}
}

func (gwd GasWantedDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	evmParams := gwd.EvmKeeper.GetParams(ctx)
	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(gwd.EvmKeeper.EthChainID(ctx))

	blockHeight := big.NewInt(ctx.BlockHeight())
	isLondon := ethCfg.IsLondon(blockHeight)

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok || !isLondon {
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
