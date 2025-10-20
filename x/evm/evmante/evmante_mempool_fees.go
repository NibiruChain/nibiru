// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ AnteStep = AnteStepSetupCtx

// Set an empty gas config so that gas payment and refund is consistent with
// Ethereum.
func AnteStepSetupCtx(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	sdb.SetCtx(
		sdb.Ctx().WithGasMeter(sdk.NewInfiniteGasMeter()).
			WithKVGasConfig(storetypes.GasConfig{}).
			WithTransientKVGasConfig(storetypes.GasConfig{}).
			WithIsEvmTx(true),
	)
	return nil
}

var _ AnteStep = AnteStepMempoolGasPrice

func AnteStepMempoolGasPrice(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	if !sdb.Ctx().IsCheckTx() && !simulate {
		return nil
	}

	minGasPrice := sdb.Ctx().MinGasPrices().AmountOf(evm.EVMBankDenom)
	baseFeeMicronibi := k.BaseFeeMicronibiPerGas(sdb.Ctx())
	baseFeeMicronibiDec := sdkmath.LegacyNewDecFromBigInt(baseFeeMicronibi)

	// if MinGasPrices is not set, skip the check
	if minGasPrice.IsZero() {
		return nil
	} else if minGasPrice.LT(baseFeeMicronibiDec) {
		minGasPrice = baseFeeMicronibiDec
	}

	baseFeeWei := evm.NativeToWei(baseFeeMicronibi)
	effectiveGasPriceDec := sdkmath.LegacyNewDecFromBigInt(
		evm.WeiToNative(msgEthTx.EffectiveGasPriceWeiPerGas(baseFeeWei)),
	)
	if effectiveGasPriceDec.LT(minGasPrice) {
		// if sdk.NewDecFromBigInt(effectiveGasPrice).LT(minGasPrice) {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInsufficientFee,
			"provided gas price < minimum local gas price (%s < %s). "+
				"Please increase the priority tip (for EIP-1559 txs) or the gas prices "+
				"(for access list or legacy txs)",
			effectiveGasPriceDec, minGasPrice,
		)
	}

	return nil
}

// ------------------------------------------------------
// OLD Implementation
// ------------------------------------------------------

var _ sdk.AnteDecorator = MempoolGasPriceDecorator{}

// MempoolGasPriceDecorator will check if the transaction's fee is at least as large
// as the mempool MinGasPrices param. If fee is too low, decorator returns error and tx
// is rejected. This applies to CheckTx only.
// If fee is high enough, then call next AnteHandler
type MempoolGasPriceDecorator struct {
	evmKeeper *EVMKeeper
}

// NewMempoolGasPriceDecorator creates a new MinGasPriceDecorator instance used only for
// Ethereum transactions.
func NewMempoolGasPriceDecorator(k *EVMKeeper) MempoolGasPriceDecorator {
	return MempoolGasPriceDecorator{
		evmKeeper: k,
	}
}

// AnteHandle ensures that the effective fee from the transaction is greater than the
// local mempool gas prices, which is defined by the  MinGasPrice (parameter) * GasLimit (tx argument).
func (d MempoolGasPriceDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// only run on CheckTx
	if !ctx.IsCheckTx() && !simulate {
		return next(ctx, tx, simulate)
	}

	minGasPrice := ctx.MinGasPrices().AmountOf(evm.EVMBankDenom)
	baseFeeMicronibi := d.evmKeeper.BaseFeeMicronibiPerGas(ctx)
	baseFeeMicronibiDec := sdkmath.LegacyNewDecFromBigInt(baseFeeMicronibi)

	// if MinGasPrices is not set, skip the check
	if minGasPrice.IsZero() {
		return next(ctx, tx, simulate)
	} else if minGasPrice.LT(baseFeeMicronibiDec) {
		minGasPrice = baseFeeMicronibiDec
	}

	msgEthTx, err := evm.RequireStandardEVMTxMsg(tx)
	if err != nil {
		return ctx, err
	}

	baseFeeWei := evm.NativeToWei(baseFeeMicronibi)
	effectiveGasPriceDec := sdkmath.LegacyNewDecFromBigInt(
		evm.WeiToNative(msgEthTx.EffectiveGasPriceWeiPerGas(baseFeeWei)),
	)
	if effectiveGasPriceDec.LT(minGasPrice) {
		// if sdk.NewDecFromBigInt(effectiveGasPrice).LT(minGasPrice) {
		return ctx, sdkioerrors.Wrapf(
			sdkerrors.ErrInsufficientFee,
			"provided gas price < minimum local gas price (%s < %s). "+
				"Please increase the priority tip (for EIP-1559 txs) or the gas prices "+
				"(for access list or legacy txs)",
			effectiveGasPriceDec, minGasPrice,
		)
	}

	return next(ctx, tx, simulate)
}
