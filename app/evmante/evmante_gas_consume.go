// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
)

// AnteDecEthGasConsume validates enough intrinsic gas for the transaction and
// gas consumption.
type AnteDecEthGasConsume struct {
	evmKeeper    EVMKeeper
	maxGasWanted uint64
}

// NewAnteDecEthGasConsume creates a new EthGasConsumeDecorator
func NewAnteDecEthGasConsume(
	k EVMKeeper,
	maxGasWanted uint64,
) AnteDecEthGasConsume {
	return AnteDecEthGasConsume{
		evmKeeper:    k,
		maxGasWanted: maxGasWanted,
	}
}

// AnteHandle validates that the Ethereum tx message has enough to cover
// intrinsic gas (during CheckTx only) and that the sender has enough balance to
// pay for the gas cost. If the balance is not sufficient, it will be attempted
// to withdraw enough staking rewards for the payment.
//
// Intrinsic gas for a transaction is the amount of gas that the transaction uses
// before the transaction is executed. The gas is a constant value plus any cost
// incurred by additional bytes of data supplied with the transaction.
//
// This AnteHandler decorator will fail if:
//   - the message is not a MsgEthereumTx
//   - sender account cannot be found
//   - transaction's gas limit is lower than the intrinsic gas
//   - user has neither enough balance nor staking rewards to deduct the transaction fees (gas_limit * gas_price)
//   - transaction or block gas meter runs out of gas
//   - sets the gas meter limit
//   - gas limit is greater than the block gas meter limit
func (anteDec AnteDecEthGasConsume) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	gasWanted := uint64(0)
	if ctx.IsReCheckTx() {
		// Then, the limit for gas consumed was already checked during CheckTx so
		// there's no need to verify it again during ReCheckTx
		//
		// Use new context with gasWanted = 0
		// Otherwise, there's an error on txmempool.postCheck (tendermint)
		// that is not bubbled up. Thus, the Tx never runs on DeliverMode
		// Error: "gas wanted -1 is negative"
		newCtx := ctx.WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted))
		return next(newCtx, tx, simulate)
	}

	evmParams := anteDec.evmKeeper.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()

	var events sdk.Events

	// Use the lowest priority of all the messages as the final one.
	minPriority := int64(math.MaxInt64)
	baseFee := anteDec.evmKeeper.GetBaseFee(ctx)

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(
				errortypes.ErrUnknownRequest,
				"invalid message type %T, expected %T",
				msg, (*evm.MsgEthereumTx)(nil),
			)
		}
		from := msgEthTx.GetFrom()

		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errors.Wrap(err, "failed to unpack tx data")
		}

		if ctx.IsCheckTx() && anteDec.maxGasWanted != 0 {
			// We can't trust the tx gas limit, because we'll refund the unused gas.
			if txData.GetGas() > anteDec.maxGasWanted {
				gasWanted += anteDec.maxGasWanted
			} else {
				gasWanted += txData.GetGas()
			}
		} else {
			gasWanted += txData.GetGas()
		}

		fees, err := keeper.VerifyFee(txData, evmDenom, baseFee, ctx.IsCheckTx())
		if err != nil {
			return ctx, errors.Wrapf(err, "failed to verify the fees")
		}

		if err = anteDec.deductFee(ctx, fees, from); err != nil {
			return ctx, err
		}

		events = append(events,
			sdk.NewEvent(
				sdk.EventTypeTx,
				sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
			),
		)

		priority := evm.GetTxPriority(txData, baseFee)

		if priority < minPriority {
			minPriority = priority
		}
	}

	ctx.EventManager().EmitEvents(events)

	blockGasLimit := eth.BlockGasLimit(ctx)

	// return error if the tx gas is greater than the block limit (max gas)

	// NOTE: it's important here to use the gas wanted instead of the gas consumed
	// from the tx gas pool. The latter only has the value so far since the
	// EthSetupContextDecorator, so it will never exceed the block gas limit.
	if gasWanted > blockGasLimit {
		return ctx, errors.Wrapf(
			errortypes.ErrOutOfGas,
			"tx gas (%d) exceeds block gas limit (%d)",
			gasWanted,
			blockGasLimit,
		)
	}

	// Set tx GasMeter with a limit of GasWanted (i.e. gas limit from the Ethereum tx).
	// The gas consumed will be then reset to the gas used by the state transition
	// in the EVM.

	// FIXME: use a custom gas configuration that doesn't add any additional gas and only
	// takes into account the gas consumed at the end of the EVM transaction.
	newCtx := ctx.
		WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted)).
		WithPriority(minPriority)

	// we know that we have enough gas on the pool to cover the intrinsic gas
	return next(newCtx, tx, simulate)
}

// deductFee checks if the fee payer has enough funds to pay for the fees and deducts them.
// If the spendable balance is not enough, it tries to claim enough staking rewards to cover the fees.
func (anteDec AnteDecEthGasConsume) deductFee(ctx sdk.Context, fees sdk.Coins, feePayer sdk.AccAddress) error {
	if fees.IsZero() {
		return nil
	}

	// If the account balance is not sufficient, try to withdraw enough staking rewards

	if err := anteDec.evmKeeper.DeductTxCostsFromUserBalance(ctx, fees, gethcommon.BytesToAddress(feePayer)); err != nil {
		return errors.Wrapf(err, "failed to deduct transaction costs from user balance")
	}
	return nil
}
