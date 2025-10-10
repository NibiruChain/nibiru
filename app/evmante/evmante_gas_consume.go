// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

// AnteDecEthGasConsume validates enough intrinsic gas for the transaction and
// gas consumption.
type AnteDecEthGasConsume struct {
	evmKeeper    *EVMKeeper
	maxGasWanted uint64
}

// // NewAnteDecEthGasConsume creates a new EthGasConsumeDecorator
// func NewAnteDecEthGasConsume(
// 	k *EVMKeeper,
// 	maxGasWanted uint64,
// ) AnteDecEthGasConsume {
// 	return AnteDecEthGasConsume{
// 		evmKeeper:    k,
// 		maxGasWanted: maxGasWanted,
// 	}
// }

// // AnteHandle validates that the Ethereum tx message has enough to cover
// // intrinsic gas (during CheckTx only) and that the sender has enough balance to
// // pay for the gas cost.
// //
// // Intrinsic gas for a transaction is the amount of gas that the transaction uses
// // before the transaction is executed. The gas is a constant value plus any cost
// // incurred by additional bytes of data supplied with the transaction.
// //
// // This AnteHandler decorator will fail if:
// //   - the message is not a MsgEthereumTx
// //   - sender account cannot be found
// //   - transaction's gas limit is lower than the intrinsic gas
// //   - user has neither enough balance nor staking rewards to deduct the transaction fees (gas_limit * gas_price)
// //   - transaction or block gas meter runs out of gas
// //   - sets the gas meter limit
// //   - gas limit is greater than the block gas meter limit
// func (anteDec AnteDecEthGasConsume) AnteHandle(
// 	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
// ) (sdk.Context, error) {
// 	gasWanted := uint64(0)
// 	if ctx.IsReCheckTx() {
// 		// Then, the limit for gas consumed was already checked during CheckTx so
// 		// there's no need to verify it again during ReCheckTx
// 		//
// 		// Use new context with gasWanted = 0
// 		// Otherwise, there's an error on txmempool.postCheck (tendermint)
// 		// that is not bubbled up. Thus, the Tx never runs on DeliverMode
// 		// Error: "gas wanted -1 is negative"
// 		newCtx := ctx.WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted))
// 		return next(newCtx, tx, simulate)
// 	}

// 	var events sdk.Events

// 	// Use the lowest priority of all the messages as the final one.
// 	minPriority := int64(math.MaxInt64)
// 	baseFeeMicronibiPerGas := anteDec.evmKeeper.BaseFeeMicronibiPerGas(ctx)

// 	msgEthTx, err := evm.RequireStandardEVMTxMsg(tx)
// 	if err != nil {
// 		return ctx, err
// 	}

// 	from := msgEthTx.GetFrom()

// 	txData, err := evm.UnpackTxData(msgEthTx.Data)
// 	if err != nil {
// 		return ctx, sdkioerrors.Wrap(err, "failed to unpack tx data")
// 	}

// 	if ctx.IsCheckTx() && anteDec.maxGasWanted != 0 {
// 		// We can't trust the tx gas limit, because we'll refund the unused gas.
// 		if txData.GetGas() > anteDec.maxGasWanted {
// 			gasWanted += anteDec.maxGasWanted
// 		} else {
// 			gasWanted += txData.GetGas()
// 		}
// 	} else {
// 		gasWanted += txData.GetGas()
// 	}

// 	fees, err := evmstate.VerifyFee(
// 		txData,
// 		baseFeeMicronibiPerGas,
// 		ctx,
// 	)
// 	if err != nil {
// 		return ctx, sdkioerrors.Wrapf(err, "failed to verify the fees")
// 	}

// 	if err = anteDec.deductFee(ctx, fees, from); err != nil {
// 		return ctx, err
// 	}

// 	events = append(events,
// 		sdk.NewEvent(
// 			sdk.EventTypeTx,
// 			sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
// 		),
// 	)

// 	priority := evm.GetTxPriority(txData, baseFeeMicronibiPerGas)

// 	if priority < minPriority {
// 		minPriority = priority
// 	}

// 	ctx.EventManager().EmitEvents(events)

// 	blockGasLimit := eth.BlockGasLimit(ctx)

// 	// return error if the tx gas is greater than the block limit (max gas)

// 	// NOTE: it's important here to use the gas wanted instead of the gas consumed
// 	// from the tx gas pool. The latter only has the value so far since the
// 	// EthSetupContextDecorator, so it will never exceed the block gas limit.
// 	if gasWanted > blockGasLimit {
// 		return ctx, sdkioerrors.Wrapf(
// 			sdkerrors.ErrOutOfGas,
// 			"tx gas (%d) exceeds block gas limit (%d)",
// 			gasWanted,
// 			blockGasLimit,
// 		)
// 	}

// 	// Set tx GasMeter with a limit of GasWanted (i.e. gas limit from the Ethereum tx).
// 	// The gas consumed will be then reset to the gas used by the state transition
// 	// in the EVM.

// 	// FIXME: use a custom gas configuration that doesn't add any additional gas and only
// 	// takes into account the gas consumed at the end of the EVM transaction.
// 	newCtx := ctx.
// 		WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted)).
// 		WithPriority(minPriority)

// 	// we know that we have enough gas on the pool to cover the intrinsic gas
// 	return next(newCtx, tx, simulate)
// }

// deductFee checks if the fee payer has enough funds to pay for the fees and
// deducts them.
// func (anteDec AnteDecEthGasConsume) deductFee(
// 	ctx sdk.Context, fees sdk.Coins, feePayer sdk.AccAddress,
// ) error {
// 	if fees.IsZero() {
// 		return nil
// 	}

// 	if err := anteDec.evmKeeper.DeductTxCostsFromUserBalance(
// 		ctx, fees, gethcommon.BytesToAddress(feePayer),
// 	); err != nil {
// 		return sdkioerrors.Wrapf(err, "failed to deduct transaction costs from user balance")
// 	}
// 	return nil
// }

var _ EvmAnteHandler = EthAnteGasConsume

type AnteOptionsEVM interface {
	GetMaxTxGasWanted() uint64
}

var _ EvmAnteHandler = EthAnteBlockGasMeter

func EthAnteBlockGasMeter(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	// return error if the tx gas is greater than the block limit (max gas)
	blockGasLimit := eth.BlockGasLimit(sdb.Ctx())

	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to unpack tx data")
	}

	var gasWanted uint64
	if sdb.Ctx().IsCheckTx() && opts.GetMaxTxGasWanted() != 0 {
		// We can't trust the tx gas limit, because we'll refund the unused gas.
		if txData.GetGas() > opts.GetMaxTxGasWanted() {
			gasWanted += opts.GetMaxTxGasWanted()
		} else {
			gasWanted += txData.GetGas()
		}
	} else {
		gasWanted += txData.GetGas()
	}

	// NOTE: it's important here to use the gas wanted instead of the gas consumed
	// from the tx gas pool. The latter only has the value so far since the
	// EthSetupContextDecorator, so it will never exceed the block gas limit.
	if gasWanted > blockGasLimit {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrOutOfGas,
			"tx gas (%d) exceeds block gas limit (%d)",
			gasWanted,
			blockGasLimit,
		)
	}

	return nil
}

func EthAnteGasConsume(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	gasWanted := uint64(0)
	if sdb.Ctx().IsReCheckTx() {
		// Then, the limit for gas consumed was already checked during CheckTx so
		// there's no need to verify it again during ReCheckTx
		//
		// Use new context with gasWanted = 0
		// Otherwise, there's an error on txmempool.postCheck (tendermint)
		// that is not bubbled up. Thus, the Tx never runs on DeliverMode
		// Error: "gas wanted -1 is negative"
		newCtx := sdb.Ctx().WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted))
		sdb.SetCtx(newCtx)
		return nil
	}

	var events sdk.Events

	// Use the lowest priority of all the messages as the final one.
	minPriority := int64(math.MaxInt64)
	baseFeeMicronibiPerGas := k.BaseFeeMicronibiPerGas(sdb.Ctx())

	from := msgEthTx.FromAddrBech32()

	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to unpack tx data")
	}

	fees, err := evmstate.VerifyFee(
		txData,
		baseFeeMicronibiPerGas,
		sdb.Ctx(),
	)
	if err != nil {
		return sdkioerrors.Wrapf(err, "failed to verify the fees")
	}

	err = func(sdb *evmstate.SDB, effFeeWei *uint256.Int, feePayer sdk.AccAddress) error {

		if fees.IsZero() {
			return nil
		}

		if err := k.DeductTxCostsFromUserBalance(
			sdb, effFeeWei, gethcommon.BytesToAddress(feePayer),
		); err != nil {
			return sdkioerrors.Wrapf(err, "failed to deduct transaction costs from user balance")
		}
		return nil
	}(sdb, fees, from)
	if err != nil {
		return err
	}

	msgEthTx.FromAddrBech32()
	events = append(events,
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, from.String()),
			evm.AttributeKeyFeePayerEvm(msgEthTx.FromAddr()),
		),
	)

	priority := evm.GetTxPriority(txData, baseFeeMicronibiPerGas)

	if priority < minPriority {
		minPriority = priority
	}

	sdb.Ctx().EventManager().EmitEvents(events)

	// Set tx GasMeter with a limit of GasWanted (i.e. gas limit from the Ethereum tx).
	// The gas consumed will be then reset to the gas used by the state transition
	// in the EVM.

	// FIXME: use a custom gas configuration that doesn't add any additional gas and only
	// takes into account the gas consumed at the end of the EVM transaction.
	newCtx := sdb.Ctx().
		WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted)).
		WithPriority(minPriority)
	sdb.SetCtx(newCtx)

	// we know that we have enough gas on the pool to cover the intrinsic gas
	return nil
}

var _ EvmAnteHandler = EthAnteFiniteGasLimitForABCIDeliverTx

// The final [EvmAnteHandler] called before the start of the Ethereum tx. This
// sets a finite gas meter with a limit so that the "BaseApp" can properly record
// the gas wanted field for [sdk.GasInfo]. This becomes the gas wanted field for
// the ABCI deliver tx result.
func EthAnteFiniteGasLimitForABCIDeliverTx(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	sdb.SetCtx(
		sdb.Ctx().
			WithGasMeter(sdk.NewGasMeter(msgEthTx.GetGas())),
	)
	return nil
}
