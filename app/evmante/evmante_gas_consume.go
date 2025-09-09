// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"fmt"
	"math"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	gastokenante "github.com/NibiruChain/nibiru/v2/x/gastoken/ante"
	gastokenkeeper "github.com/NibiruChain/nibiru/v2/x/gastoken/keeper"
	gastokentypes "github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

// AnteDecEthGasConsume validates enough intrinsic gas for the transaction and
// gas consumption.
type AnteDecEthGasConsume struct {
	evmKeeper      *EVMKeeper
	accountKeeper  evm.AccountKeeper
	gasTokenKeeper *gastokenkeeper.Keeper
	maxGasWanted   uint64
}

// NewAnteDecEthGasConsume creates a new EthGasConsumeDecorator
func NewAnteDecEthGasConsume(
	k *EVMKeeper,
	ak evm.AccountKeeper,
	gtk *gastokenkeeper.Keeper,
	maxGasWanted uint64,
) AnteDecEthGasConsume {
	return AnteDecEthGasConsume{
		evmKeeper:      k,
		accountKeeper:  ak,
		gasTokenKeeper: gtk,
		maxGasWanted:   maxGasWanted,
	}
}

// AnteHandle validates that the Ethereum tx message has enough to cover
// intrinsic gas (during CheckTx only) and that the sender has enough balance to
// pay for the gas cost.
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

	var events sdk.Events

	// Use the lowest priority of all the messages as the final one.
	minPriority := int64(math.MaxInt64)
	baseFeeMicronibiPerGas := anteDec.evmKeeper.BaseFeeMicronibiPerGas(ctx)

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, sdkioerrors.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"invalid message type %T, expected %T",
				msg, (*evm.MsgEthereumTx)(nil),
			)
		}
		from := msgEthTx.GetFrom()

		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, sdkioerrors.Wrap(err, "failed to unpack tx data")
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

		fees, err := keeper.VerifyFee(
			txData,
			baseFeeMicronibiPerGas,
			ctx,
		)
		if err != nil {
			return ctx, sdkioerrors.Wrapf(err, "failed to verify the fees")
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

		priority := evm.GetTxPriority(txData, baseFeeMicronibiPerGas)

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
		return ctx, sdkioerrors.Wrapf(
			sdkerrors.ErrOutOfGas,
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

// deductFee checks if the fee payer has enough funds to pay for the fees and
// deducts them.
func (anteDec AnteDecEthGasConsume) deductFee(
	ctx sdk.Context, fees sdk.Coins, feePayer sdk.AccAddress,
) error {
	if fees.IsZero() {
		return nil
	}

	feePayerAddr := gethcommon.BytesToAddress(feePayer)

	gasTokenUsed := ctx.Value(gasTokenUsedKey)
	useWNibiVal := ctx.Value(useWNibiKey)
	useWNibi, _ := useWNibiVal.(bool)

	feeCollector := eth.NibiruAddrToEthAddr(anteDec.accountKeeper.GetModuleAddress(gastokentypes.ModuleName))
	switch {
	case gasTokenUsed != nil:
		gasTokenStr, ok := gasTokenUsed.(string)
		if !ok {
			return fmt.Errorf("gas token address in context is not a string: %v", gasTokenUsed)
		}

		gasTokenAddress := gethcommon.HexToAddress(gasTokenStr)
		amtVal := ctx.Value(gasTokenAmountKey)
		gasTokenAmount, ok := amtVal.(*big.Int)
		if !ok || gasTokenAmount == nil || gasTokenAmount.Sign() <= 0 {
			return fmt.Errorf("invalid gas token amount in context: %v", amtVal)
		}
		params, err := anteDec.gasTokenKeeper.GetParams(ctx)
		if err != nil {
			return sdkioerrors.Wrapf(err, "failed to get gastoken params")
		}
		nonce := anteDec.evmKeeper.GetAccNonce(ctx, feePayerAddr)
		wnibi := params.WnibiAddress
		if err := gastokenante.SwapFeeToken(ctx, anteDec.evmKeeper, anteDec.accountKeeper, anteDec.gasTokenKeeper, feePayerAddr, gasTokenAddress, feeCollector, gasTokenAmount, evm.NativeToWei(fees[0].Amount.BigInt())); err != nil {
			return sdkioerrors.Wrapf(err, "failed to swap gas token %s", gasTokenAddress)
		}

		if err := gastokenante.WithdrawFeeToken(ctx, anteDec.evmKeeper, anteDec.accountKeeper, gethcommon.HexToAddress(wnibi), feeCollector, big.NewInt(0)); err != nil {
			return sdkioerrors.Wrapf(err, "failed to withdraw base token %s", gethcommon.HexToAddress(wnibi))
		}

		acc := anteDec.accountKeeper.GetAccount(ctx, feePayer)
		if err := acc.SetSequence(nonce); err != nil {
			return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
		}

		anteDec.accountKeeper.SetAccount(ctx, acc)
		return nil

	case useWNibi:
		params, err := anteDec.gasTokenKeeper.GetParams(ctx)
		if err != nil {
			return sdkioerrors.Wrapf(err, "failed to get gastoken params")
		}
		wnibi := params.WnibiAddress

		wnibiBal, err := anteDec.evmKeeper.GetErc20Balance(ctx, feePayerAddr, gethcommon.HexToAddress(wnibi))
		if err != nil {
			return sdkioerrors.Wrapf(err, "failed to get WNIBI balance for account %s", feePayerAddr)
		}

		nonce := anteDec.evmKeeper.GetAccNonce(ctx, feePayerAddr)

		feesAmount := fees[0].Amount
		if wnibiBal.Cmp(evm.NativeToWei(feesAmount.BigInt())) < 0 {
			return sdkerrors.ErrInsufficientFee.Wrapf(
				"insufficient WNIBI for fees: have %s want %s",
				wnibiBal.String(), evm.NativeToWei(feesAmount.BigInt()).String(),
			)
		}

		// If the user has enough WNIBI, just deduct in WNIBI
		err = anteDec.evmKeeper.Erc20Transfer(ctx, gethcommon.HexToAddress(wnibi), feePayerAddr, feeCollector, evm.NativeToWei(feesAmount.BigInt()))
		if err != nil {
			return sdkioerrors.Wrapf(err, "failed to transfer WNIBI from %s to fee collector", feePayerAddr)
		}

		if err := gastokenante.WithdrawFeeToken(ctx, anteDec.evmKeeper, anteDec.accountKeeper, gethcommon.HexToAddress(wnibi), feeCollector, big.NewInt(0)); err != nil {
			return sdkioerrors.Wrapf(err, "failed to withdraw base token %s", gethcommon.HexToAddress(wnibi))
		}

		acc := anteDec.accountKeeper.GetAccount(ctx, feePayer)
		if err := acc.SetSequence(nonce); err != nil {
			return sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce)
		}

		anteDec.accountKeeper.SetAccount(ctx, acc)
		return nil
	default:
		if err := anteDec.evmKeeper.DeductTxCostsFromUserBalance(
			ctx, fees, gethcommon.BytesToAddress(feePayer),
		); err != nil {
			return sdkioerrors.Wrapf(err, "failed to deduct transaction costs from user balance")
		}
		return nil
	}
}
