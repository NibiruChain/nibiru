package evmante

// Copyright (c) 2023-2024 Nibi, Inc.
//
// evmante_gas_consume.go: EVM gas consumption ante handlers. This file manages
// the deductio of fees and validation of tx and block gas limits.

import (
	"fmt"
	"log"
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

var _ AnteStep = AnteStepGasWanted

type AnteOptionsEVM interface {
	GetMaxTxGasWanted() uint64
}

var _ AnteStep = AnteStepBlockGasMeter

// AnteStepBlockGasMeter validates that the transaction gas limit does not exceed
// the block gas limit, ensuring each the tx doesn't consume more gas than the
// block can accommodate.
//
// This handler will fail if:
//   - the transaction gas wanted exceeds the block gas limit
//
// The block gas limit is determined by the consensus parameters and represents
// the maximum total gas that can be consumed by all transactions in a single
// block.
func AnteStepBlockGasMeter(
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
		// We can't trust the tx gas limit during CheckTx, because we'll refund the unused gas.
		// The maxGasWanted parameter provides a cap to prevent excessive gas consumption
		// during the CheckTx phase, while still allowing the full gas limit during execution.
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
	// We use gasWanted (the transaction's declared gas limit) rather than gasConsumed
	// (actual gas used so far) to ensure proper block gas limit enforcement.
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

// AnteStepDeductGas deducts the transaction fees from the sender's account
// balance. This handler calculates the effective fee based on the transaction
// data and base fee, then deducts the fee from the sender's account in the SDB
// state.
//
// This handler will fail if:
//   - the sender has insufficient balance to pay the transaction fees
//   - fee verification fails due to invalid transaction parameters
//
// The deducted fees are transferred to the fee collector account and a fee event
// is emitted to the context event manager.
func AnteStepDeductGas(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	from := msgEthTx.FromAddrBech32()
	baseFeeMicronibiPerGas := k.BaseFeeMicronibiPerGas(sdb.Ctx())
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

	fmt.Printf("EthAnteDeductGas Pre: sdb.GetBalance(msgEthTx.FromAddr()): %s\n", sdb.GetBalance(msgEthTx.FromAddr()))
	log.Printf("EthAnteDeductGas Pre: sdb.GetBalance(evm.FEE_COLLECTOR_ADDR): %s\n", sdb.GetBalance(evm.FEE_COLLECTOR_ADDR))
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
	sdb.Ctx().EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
		sdk.NewAttribute(sdk.AttributeKeyFeePayer, from.String()),
		evm.AttributeKeyFeePayerEvm(msgEthTx.FromAddr()),
	))

	fmt.Printf("EthAnteDeductGas Post: sdb.GetBalance(msgEthTx.FromAddr()): %s\n", sdb.GetBalance(msgEthTx.FromAddr()))
	log.Printf("EthAnteDeductGas Post: sdb.GetBalance(evm.FEE_COLLECTOR_ADDR): %s\n", sdb.GetBalance(evm.FEE_COLLECTOR_ADDR))

	return nil
}

// AnteStepGasWanted sets up the transaction gas meter with the appropriate gas
// limit and priority for the Ethereum transaction. This handler manages gas
// consumption tracking during CheckTx and ReCheckTx phases.
//
// During CheckTx: Sets up an infinite gas meter with the transaction's gas limit
// During ReCheckTx: Uses a gas meter with 0 limit to avoid gas calculation
// errors
//
// This handler will fail if:
//   - transaction data cannot be unpacked
//   - gas limit validation fails
//
// The handler sets the context gas meter to track gas consumption and
// establishes transaction priority based on the effective gas price.
func AnteStepGasWanted(
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
		// Use new context with gasWanted = 0 to avoid gas calculation errors
		// Otherwise, there's an error on txmempool.postCheck (tendermint)
		// that is not bubbled up. Thus, the Tx never runs on DeliverMode
		// Error: "gas wanted -1 is negative"
		// We set an infinite gas meter with 0 limit for ReCheckTx to prevent
		// gas calculation issues while maintaining the gas tracking infrastructure.
		newCtx := sdb.Ctx().WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted))
		sdb.SetCtx(newCtx)
		return nil
	}

	// Use the lowest priority of all the messages as the final one.
	minPriority := int64(math.MaxInt64)
	baseFeeMicronibiPerGas := k.BaseFeeMicronibiPerGas(sdb.Ctx())

	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to unpack tx data")
	}

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

	priority := evm.GetTxPriority(txData, baseFeeMicronibiPerGas)

	if priority < minPriority {
		minPriority = priority
	}

	// Set the context gas meter with the transaction's gas limit.
	// The infinite gas meter with limit allows tracking gas consumption up to the
	// specified limit without hard failures during the ante handler phase.
	//
	// Gas consumption follows a two-phase approach:
	// 1. Ante handlers use infinite gas meters to avoid premature failures
	// 2. EVM execution tracks actual gas consumption separately via gasRemaining
	// 3. Only the final consumed amount is recorded via SafeConsumeGas
	//
	// This design ensures that Cosmos gas accounting reflects actual EVM gas usage
	// without double-counting or premature gas failures during validation.
	newCtx := sdb.Ctx().
		WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted)).
		WithPriority(minPriority)
	sdb.SetCtx(newCtx)

	// we know that we have enough gas on the pool to cover the intrinsic gas
	return nil
}

var _ AnteStep = AnteStepFiniteGasLimitForABCIDeliverTx

// AnteStepFiniteGasLimitForABCIDeliverTx is the final EvmAnteHandler called
// before the start of the Ethereum transaction execution. This handler sets a
// finite gas meter with the transaction's gas limit so that the BaseApp can
// properly record the gas wanted field for sdk.GasInfo.
//
// This handler is essential for ABCI deliver tx results as it establishes the
// final gas meter that will be used during transaction execution and provides
// accurate gas consumption metrics to the consensus layer.
//
// The finite gas meter ensures that the transaction cannot consume more gas than
// specified in the transaction's gas limit, providing a hard upper bound for gas
// consumption during EVM execution.
func AnteStepFiniteGasLimitForABCIDeliverTx(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	// Set a finite gas meter with the transaction's gas limit for final
	// execution. This finite gas meter provides the hard upper bound for gas
	// consumption during EVM execution and ensures accurate gas reporting to the
	// consensus layer.
	sdb.SetCtx(
		sdb.Ctx().
			WithGasMeter(sdk.NewGasMeter(msgEthTx.GetGas())),
	)
	return nil
}
