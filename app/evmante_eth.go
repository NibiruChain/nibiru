// Copyright (c) 2023-2024 Nibi, Inc.
package app

import (
	"math"
	"math/big"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
	"github.com/NibiruChain/nibiru/x/evm/statedb"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
)

var (
	_ sdk.AnteDecorator = (*EthGasConsumeDecorator)(nil)
	_ sdk.AnteDecorator = (*EthAccountVerificationDecorator)(nil)
)

// EthAccountVerificationDecorator validates an account balance checks
type EthAccountVerificationDecorator struct {
	AppKeepers
}

// NewEthAccountVerificationDecorator creates a new EthAccountVerificationDecorator
func NewEthAccountVerificationDecorator(k AppKeepers) EthAccountVerificationDecorator {
	return EthAccountVerificationDecorator{
		AppKeepers: k,
	}
}

// AnteHandle validates checks that the sender balance is greater than the total transaction cost.
// The account will be set to store if it doesn't exist, i.e. cannot be found on store.
// This AnteHandler decorator will fail if:
// - any of the msgs is not a MsgEthereumTx
// - from address is empty
// - account balance is lower than the transaction cost
func (avd EthAccountVerificationDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() {
		return next(ctx, tx, simulate)
	}

	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil))
		}

		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errors.Wrapf(err, "failed to unpack tx data any for tx %d", i)
		}

		// sender address should be in the tx cache from the previous AnteHandle call
		from := msgEthTx.GetFrom()
		if from.Empty() {
			return ctx, errors.Wrap(errortypes.ErrInvalidAddress, "from address cannot be empty")
		}

		// check whether the sender address is EOA
		fromAddr := gethcommon.BytesToAddress(from)
		acct := avd.EvmKeeper.GetAccount(ctx, fromAddr)

		if acct == nil {
			acc := avd.AccountKeeper.NewAccountWithAddress(ctx, from)
			avd.AccountKeeper.SetAccount(ctx, acc)
			acct = statedb.NewEmptyAccount()
		} else if acct.IsContract() {
			return ctx, errors.Wrapf(errortypes.ErrInvalidType,
				"the sender is not EOA: address %s, codeHash <%s>", fromAddr, acct.CodeHash)
		}

		if err := keeper.CheckSenderBalance(sdkmath.NewIntFromBigInt(acct.Balance), txData); err != nil {
			return ctx, errors.Wrap(err, "failed to check sender balance")
		}
	}
	return next(ctx, tx, simulate)
}

// EthGasConsumeDecorator validates enough intrinsic gas for the transaction and
// gas consumption.
type EthGasConsumeDecorator struct {
	AppKeepers
	// bankKeeper         anteutils.BankKeeper
	// distributionKeeper anteutils.DistributionKeeper
	// evmKeeper          EVMKeeper
	// stakingKeeper      anteutils.StakingKeeper
	maxGasWanted uint64
}

// NewEthGasConsumeDecorator creates a new EthGasConsumeDecorator
func NewEthGasConsumeDecorator(
	keepers AppKeepers,
	maxGasWanted uint64,
) EthGasConsumeDecorator {
	return EthGasConsumeDecorator{
		AppKeepers:   keepers,
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
func (egcd EthGasConsumeDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	gasWanted := uint64(0)
	// gas consumption limit already checked during CheckTx so there's no need to
	// verify it again during ReCheckTx
	if ctx.IsReCheckTx() {
		// Use new context with gasWanted = 0
		// Otherwise, there's an error on txmempool.postCheck (tendermint)
		// that is not bubbled up. Thus, the Tx never runs on DeliverMode
		// Error: "gas wanted -1 is negative"
		newCtx := ctx.WithGasMeter(eth.NewInfiniteGasMeterWithLimit(gasWanted))
		return next(newCtx, tx, simulate)
	}

	evmParams := egcd.EvmKeeper.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(egcd.EvmKeeper.EthChainID(ctx))

	blockHeight := big.NewInt(ctx.BlockHeight())
	homestead := ethCfg.IsHomestead(blockHeight)
	istanbul := ethCfg.IsIstanbul(blockHeight)
	var events sdk.Events

	// Use the lowest priority of all the messages as the final one.
	minPriority := int64(math.MaxInt64)
	baseFee := egcd.EvmKeeper.GetBaseFee(ctx, ethCfg)

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

		if ctx.IsCheckTx() && egcd.maxGasWanted != 0 {
			// We can't trust the tx gas limit, because we'll refund the unused gas.
			if txData.GetGas() > egcd.maxGasWanted {
				gasWanted += egcd.maxGasWanted
			} else {
				gasWanted += txData.GetGas()
			}
		} else {
			gasWanted += txData.GetGas()
		}

		fees, err := keeper.VerifyFee(txData, evmDenom, baseFee, homestead, istanbul, ctx.IsCheckTx())
		if err != nil {
			return ctx, errors.Wrapf(err, "failed to verify the fees")
		}

		if err = egcd.deductFee(ctx, fees, from); err != nil {
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
func (egcd EthGasConsumeDecorator) deductFee(ctx sdk.Context, fees sdk.Coins, feePayer sdk.AccAddress) error {
	if fees.IsZero() {
		return nil
	}

	// If the account balance is not sufficient, try to withdraw enough staking rewards

	if err := egcd.EvmKeeper.DeductTxCostsFromUserBalance(ctx, fees, gethcommon.BytesToAddress(feePayer)); err != nil {
		return errors.Wrapf(err, "failed to deduct transaction costs from user balance")
	}
	return nil
}

// CanTransferDecorator checks if the sender is allowed to transfer funds according to the EVM block
// context rules.
type CanTransferDecorator struct {
	AppKeepers
}

// NewCanTransferDecorator creates a new CanTransferDecorator instance.
func NewCanTransferDecorator(k AppKeepers) CanTransferDecorator {
	return CanTransferDecorator{
		AppKeepers: k,
	}
}

// AnteHandle creates an EVM from the message and calls the BlockContext CanTransfer function to
// see if the address can execute the transaction.
func (ctd CanTransferDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	params := ctd.EvmKeeper.GetParams(ctx)
	ethCfg := params.ChainConfig.EthereumConfig(ctd.EvmKeeper.EthChainID(ctx))
	signer := gethcore.MakeSigner(ethCfg, big.NewInt(ctx.BlockHeight()))

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil))
		}

		baseFee := ctd.EvmKeeper.GetBaseFee(ctx, ethCfg)

		coreMsg, err := msgEthTx.AsMessage(signer, baseFee)
		if err != nil {
			return ctx, errors.Wrapf(
				err,
				"failed to create an ethereum core.Message from signer %T", signer,
			)
		}

		if evm.IsLondon(ethCfg, ctx.BlockHeight()) {
			if baseFee == nil {
				return ctx, errors.Wrap(
					evm.ErrInvalidBaseFee,
					"base fee is supported but evm block context value is nil",
				)
			}
			if coreMsg.GasFeeCap().Cmp(baseFee) < 0 {
				return ctx, errors.Wrapf(
					errortypes.ErrInsufficientFee,
					"max fee per gas less than block base fee (%s < %s)",
					coreMsg.GasFeeCap(), baseFee,
				)
			}
		}

		// NOTE: pass in an empty coinbase address and nil tracer as we don't need them for the check below
		cfg := &statedb.EVMConfig{
			ChainConfig: ethCfg,
			Params:      params,
			CoinBase:    gethcommon.Address{},
			BaseFee:     baseFee,
		}

		stateDB := statedb.New(ctx, &ctd.EvmKeeper, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes())))
		evm := ctd.EvmKeeper.NewEVM(ctx, coreMsg, cfg, evm.NewNoOpTracer(), stateDB)

		// check that caller has enough balance to cover asset transfer for **topmost** call
		// NOTE: here the gas consumed is from the context with the infinite gas meter
		if coreMsg.Value().Sign() > 0 && !evm.Context.CanTransfer(stateDB, coreMsg.From(), coreMsg.Value()) {
			return ctx, errors.Wrapf(
				errortypes.ErrInsufficientFunds,
				"failed to transfer %s from address %s using the EVM block context transfer function",
				coreMsg.Value(),
				coreMsg.From(),
			)
		}
	}

	return next(ctx, tx, simulate)
}

// EthIncrementSenderSequenceDecorator increments the sequence of the signers.
type EthIncrementSenderSequenceDecorator struct {
	AppKeepers
}

// NewEthIncrementSenderSequenceDecorator creates a new EthIncrementSenderSequenceDecorator.
func NewEthIncrementSenderSequenceDecorator(k AppKeepers) EthIncrementSenderSequenceDecorator {
	return EthIncrementSenderSequenceDecorator{
		AppKeepers: k,
	}
}

// AnteHandle handles incrementing the sequence of the signer (i.e. sender). If the transaction is a
// contract creation, the nonce will be incremented during the transaction execution and not within
// this AnteHandler decorator.
func (issd EthIncrementSenderSequenceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, errors.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil))
		}

		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errors.Wrap(err, "failed to unpack tx data")
		}

		// increase sequence of sender
		acc := issd.AccountKeeper.GetAccount(ctx, msgEthTx.GetFrom())
		if acc == nil {
			return ctx, errors.Wrapf(
				errortypes.ErrUnknownAddress,
				"account %s is nil", gethcommon.BytesToAddress(msgEthTx.GetFrom().Bytes()),
			)
		}
		nonce := acc.GetSequence()

		// we merged the nonce verification to nonce increment, so when tx includes multiple messages
		// with same sender, they'll be accepted.
		if txData.GetNonce() != nonce {
			return ctx, errors.Wrapf(
				errortypes.ErrInvalidSequence,
				"invalid nonce; got %d, expected %d", txData.GetNonce(), nonce,
			)
		}

		if err := acc.SetSequence(nonce + 1); err != nil {
			return ctx, errors.Wrapf(err, "failed to set sequence to %d", acc.GetSequence()+1)
		}

		issd.AccountKeeper.SetAccount(ctx, acc)
	}

	return next(ctx, tx, simulate)
}
