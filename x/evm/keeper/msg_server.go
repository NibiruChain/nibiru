package keeper

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"

	sdkioerrors "cosmossdk.io/errors"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

var _ evm.MsgServer = &Keeper{}

func (k *Keeper) EthereumTx(
	goCtx context.Context, txMsg *evm.MsgEthereumTx,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	// This is a `defer` pattern to add behavior that runs in the case that the
	// error is non-nil, creating a concise way to add extra information.
	defer func() {
		if err != nil {
			err = fmt.Errorf("EthereumTx error: %w", err)
		}
	}()

	if err := txMsg.ValidateBasic(); err != nil {
		return evmResp, sdkioerrors.Wrap(err, "EthereumTx validate basic failed")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	tx := txMsg.AsTransaction()
	txConfig := k.TxConfig(ctx, tx.Hash())
	evmCfg := k.GetEVMConfig(ctx)

	// get the signer according to the chain rules from the config and block height
	evmMsg, err := core.TransactionToMessage(
		tx, gethcore.NewLondonSigner(evmCfg.ChainConfig.ChainID), evmCfg.BaseFeeWei,
	)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to convert ethereum transaction as core message")
	}

	// ApplyEvmMsg - Perform the EVM State transition
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, txConfig)
	}
	defer func() {
		k.Bank.StateDB = nil
	}()
	evmObj := k.NewEVM(ctx, *evmMsg, evmCfg, nil /*tracer*/, stateDB)

	var applyErr error
	evmResp, applyErr = k.ApplyEvmMsg(
		ctx,
		*evmMsg,
		evmObj,
		evm.COMMIT_ETH_TX, /*commit*/
		txConfig.TxHash,
	)

	if applyErr != nil {
		if evmResp == nil {
			// Consensus error - return immediately, skipping the
			// "evm.SafeConsumeGas" call we do for
			ctx.WithLastErrApplyEvmMsg(applyErr)
			return nil, sdkioerrors.Wrap(applyErr, "consensus error in ethereum message")
		} else {
			// Execution error - log but continue processing
			ctx.WithLastErrApplyEvmMsg(applyErr)
		}
	}

	if evmResp != nil {
		gasErr := evm.SafeConsumeGas(ctx, evmResp.GasUsed, "execute EthereumTx")
		if gasErr != nil {
			return nil, gasErr
		}
	}

	k.updateBlockBloom(ctx, evmResp, uint64(txConfig.LogIndex))

	// refund gas in order to match the Ethereum gas consumption instead of the
	// default SDK one.
	refundGas := uint64(0)
	if evmMsg.GasLimit > evmResp.GasUsed {
		refundGas = evmMsg.GasLimit - evmResp.GasUsed
	}
	weiPerGas := txMsg.EffectiveGasPriceWeiPerGas(evmCfg.BaseFeeWei)
	if err = k.RefundGas(ctx, evmMsg.From, refundGas, weiPerGas); err != nil {
		return nil, sdkioerrors.Wrapf(err, "error refunding leftover gas to sender %s", evmMsg.From)
	}

	err = k.EmitEthereumTxEvents(ctx, tx.To(), tx.Type(), *evmMsg, evmResp)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "error emitting ethereum tx events")
	}

	err = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmResp.Logs})
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "error emitting tx log event")
	}

	k.EvmState.BlockTxIndex.Set(ctx, uint64(txConfig.TxIndex)+1)

	if evmResp.Failed() && ctx.LastErrApplyEvmMsg() != nil {
		evmResp.VmError = fmt.Sprintf(
			"%s: %s",
			evmResp.VmError,
			ctx.LastErrApplyEvmMsg(),
		)
	}
	return evmResp, nil
}

// NewEVM generates a go-ethereum VM.
//
// Args:
//   - ctx: Consensus and KV store info for the current block.
//   - msg: Ethereum message sent to a contract
//   - cfg: Encapsulates params required to construct an EVM.
//   - tracer: Collects execution traces for EVM transaction logging.
//   - stateDB: Holds the EVM state.
func (k *Keeper) NewEVM(
	ctx sdk.Context,
	msg core.Message,
	evmCfg statedb.EVMConfig,
	tracer *tracing.Hooks,
	stateDB vm.StateDB,
) (evmObj *vm.EVM) {
	pseudoRandomBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(pseudoRandomBytes, uint64(ctx.BlockHeader().Time.UnixNano()))
	pseudoRandom := crypto.Keccak256Hash(append(pseudoRandomBytes, ctx.BlockHeader().LastCommitHash...))

	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     k.GetHashFn(ctx),
		Coinbase:    evmCfg.BlockCoinbase,
		GasLimit:    eth.BlockGasLimit(ctx),
		BlockNumber: big.NewInt(ctx.BlockHeight()),
		Time:        evm.ParseBlockTimeUnixU64(ctx),
		Difficulty:  big.NewInt(0), // unused. Only required in PoW context
		BaseFee:     evmCfg.BaseFeeWei,
		Random:      &pseudoRandom,
	}

	txCtx := core.NewEVMTxContext(&msg)
	if tracer == nil {
		// Return a default tracer (*[tracing.Hooks]) based on current keeper state
		tracer = evm.NewTracer(k.tracer, msg, evmCfg.ChainConfig, ctx.BlockHeight())
	}
	vmConfig := k.VMConfig(ctx, &evmCfg, tracer)
	evmObj = vm.NewEVM(blockCtx, txCtx, stateDB, evmCfg.ChainConfig, vmConfig)
	evmObj.AccessEvents = state.NewAccessEvents(nil) // prevents nil pointers on access
	return evmObj
}

// GetHashFn implements [vm.GetHashFunc] for the [vm.EVM] object. It handles 3 cases:
//  1. The requested height matches the current height from context (and thus same epoch number)
//  2. The requested height is from a previous height from the same chain epoch
//  3. The requested height is from a height greater than the latest one
func (k Keeper) GetHashFn(ctx sdk.Context) vm.GetHashFunc {
	return func(height uint64) gethcommon.Hash {
		h, err := eth.SafeInt64(height)

		switch {
		case err != nil:
			k.Logger(ctx).Error("failed to cast height to int64", "error", err.Error())
			return gethcommon.Hash{}

		case ctx.BlockHeight() == h:
			// Case 1: The requested height matches the one from the context, so
			// we can retrieve the header hash directly from the context. Note:
			// The headerHash is only set at begin block, it will be nil in case
			// of a query context
			headerHash := ctx.HeaderHash()
			if len(headerHash) != 0 {
				return gethcommon.BytesToHash(headerHash)
			}

			// only recompute the hash if not set (eg: checkTxState)
			contextBlockHeader := ctx.BlockHeader()
			header, err := cmttypes.HeaderFromProto(&contextBlockHeader)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err.Error())
				return gethcommon.Hash{}
			}

			headerHash = header.Hash()
			return gethcommon.BytesToHash(headerHash)

		case ctx.BlockHeight() > h:
			// Case 2: if the chain is not the current height we need to retrieve
			// the hash from the store for the current chain epoch. This only
			// applies if the current height is greater than the requested
			// height.
			histInfo, found := k.stakingKeeper.GetHistoricalInfo(ctx, h)
			if !found {
				k.Logger(ctx).Debug("historical info not found", "height", h)
				return gethcommon.Hash{}
			}

			header, err := cmttypes.HeaderFromProto(&histInfo.Header)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err.Error())
				return gethcommon.Hash{}
			}

			return gethcommon.BytesToHash(header.Hash())
		default:
			// Case 3: heights greater than the current one returns an empty hash.
			return gethcommon.Hash{}
		}
	}
}

// ApplyEvmMsg computes the new state by applying the given message against the
// existing state. If the message fails, the VM execution error with the reason
// will be returned to the client and the transaction won't be committed to the
// store.
//
// ## Reverted state
//
// The snapshot and rollback are supported by the `statedb.StateDB`.
//
// ## Different Callers
//
// It's called in three scenarios:
// 1. `ApplyTransaction`, in the transaction processing flow.
// 2. `EthCall/EthEstimateGas` grpc query handler.
// 3. Called by other native modules directly.
//
// ## Prechecks and Preprocessing
//
// All relevant state transition prechecks for the MsgEthereumTx are performed on the AnteHandler,
// prior to running the transaction against the state. The prechecks run are the following:
//
// 1. the nonce of the message caller is correct
// 2. caller has enough balance to cover transaction fee(gaslimit * gasprice)
// 3. the amount of gas required is available in the block
// 4. the purchased gas is enough to cover intrinsic usage
// 5. there is no overflow when calculating intrinsic gas
// 6. caller has enough balance to cover asset transfer for **topmost** call
//
// The preprocessing steps performed by the AnteHandler are:
//
// 1. set up the initial access list
//
// ## Tracer parameter
//
// It should be a `vm.Tracer` object or nil, if pass `nil`, it'll create a
// default one based on keeper options.
//
// ## Commit parameter
//
// If commit is true, the `StateDB` will be committed, otherwise discarded.
//
// ## fullRefundLeftoverGas parameter
//
// For internal calls like funtokens, user does not specify gas limit explicitly.
// In this case we don't apply any caps for refund and refund 100%
func (k *Keeper) ApplyEvmMsg(
	ctx sdk.Context,
	msg core.Message,
	evmObj *vm.EVM,
	commit bool,
	txHash gethcommon.Hash,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	var (
		contractCreation = msg.To == nil
		rules            = evmObj.ChainConfig().Rules(
			big.NewInt(ctx.BlockHeight()), false, evm.ParseBlockTimeUnixU64(ctx),
		)
		// gasRemaining represents a running tally of remaining gas
		// available for EVM execution. Gas remaining starts starts at
		// the [core.Message].GasLimit and is progressively reduced by:
		//
		// 1. Intrinsic gas costs (base transaction fees, data payload costs)
		// 2. Actual EVM operation execution costs
		// 3. Potential gas refunds
		//
		// It determines how much computational work can be performed before the transaction
		// runs out of gas, with unused gas potentially being refunded to the sender.
		gasRemaining = msg.GasLimit
		tracer       = evmObj.Config.Tracer
		evmStateDB   = evmObj.StateDB.(*statedb.StateDB) // retains doc comments
	)

	// Required: Allow the tracer to capture tx level events pertaining to gas consumption.
	if tracer != nil {
		// Formerly: evmObj.Config.Tracer.CaptureTxStart in geth v1.10
		if tracer.OnTxStart != nil {
			ethTx := gasRemainingTxPartial(msg.GasLimit)
			tracer.OnTxStart(
				evmObj.GetVMContext(),
				ethTx,
				msg.From,
			)
		}
		// Formerly: evmObj.Config.Tracer.CaptureTxEnd in geth v1.10
		if tracer.OnTxEnd != nil {
			defer func() {
				localEvmResp := new(evm.MsgEthereumTxResponse)
				if evmResp != nil {
					localEvmResp = evmResp
				}
				tracer.OnTxEnd(&gethcore.Receipt{
					GasUsed: localEvmResp.GasUsed,
					TxHash:  txHash,
				}, err)
			}()
		}
	}

	intrinsicGasCost, err := core.IntrinsicGas(
		msg.Data, msg.AccessList,
		contractCreation,
		rules.IsHomestead,
		rules.IsIstanbul,
		rules.IsShanghai,
	)
	if err != nil {
		// should have already been checked on Ante Handler
		return nil, sdkioerrors.Wrap(err, "ApplyEvmMsg: intrinsic gas overflowed")
	}

	// Check if the provided gas in the message is enough to cover the intrinsic
	// gas, the base gas cost before execution occurs (gethparams.TxGas, contract
	// creation, and cost per byte of the data payload).
	//
	// Should check again even if it is checked on Ante Handler, because eth_call
	// don't go through Ante Handler.
	if gasRemaining < intrinsicGasCost {
		// eth_estimateGas will check for this exact error
		return nil, fmt.Errorf(
			"ApplyEvmMsg: %s: %s: provided msg.Gas (%d) is less than intrinsic gas cost (%d)",
			vm.ErrOutOfGas, core.ErrIntrinsicGas, gasRemaining, intrinsicGasCost,
		)
	}
	if tracer != nil && tracer.OnGasChange != nil {
		tracer.OnGasChange(
			gasRemaining, gasRemaining-intrinsicGasCost, tracing.GasChangeTxIntrinsicGas)
	}
	gasRemaining -= intrinsicGasCost

	if rules.IsEIP4762 {
		evmObj.AccessEvents.AddTxOrigin(msg.From)
		if dest := msg.To; dest != nil {
			evmObj.AccessEvents.AddTxDestination(
				*dest, msg.Value.Sign() != 0,
			)
		}
	}

	msgWei, err := ParseWeiAsMultipleOfMicronibi(msg.Value)
	if err != nil {
		return nil, sdkioerrors.Wrapf(err, "ApplyEvmMsg: invalid wei amount %s", msg.Value)
	}

	// access list preparation is moved from ante handler to here, because it's
	// needed when `ApplyMessage` is called under contexts where ante handlers
	// are not run, for example `eth_call` and `eth_estimateGas`.
	evmStateDB.Prepare(
		rules,
		msg.From,                // sender
		evmObj.Context.Coinbase, // coinbase
		msg.To,
		evm.PRECOMPILE_ADDRS,
		msg.AccessList, // accessList
	)

	// take over the nonce management from evm:
	// - reset sender's nonce to msg.Nonce() before calling evm.
	// - increase sender's nonce by one no matter the result.
	evmStateDB.SetNonce(msg.From, msg.Nonce)

	var (
		returnBz []byte
		// vmErr: VM errors do not affect consensus and therefore are not assigned to "err"
		vmErr error
	)
	if contractCreation {
		returnBz, _, gasRemaining, vmErr = evmObj.Create(
			vm.AccountRef(msg.From),
			msg.Data,
			gasRemaining,
			msgWei,
		)
	} else {
		returnBz, gasRemaining, vmErr = evmObj.Call(
			vm.AccountRef(msg.From),
			*msg.To,
			msg.Data,
			gasRemaining,
			msgWei,
		)
	}
	// Increment nonce after processing the message
	evmStateDB.SetNonce(msg.From, msg.Nonce+1)

	// EVM execution error needs to be available for the JSON-RPC client
	var vmError string
	if vmErr != nil {
		vmError = vmErr.Error()
	}

	// process gas refunds (we refund a portion of the unused gas)
	gasUsed := msg.GasLimit - gasRemaining
	// please see https://eips.ethereum.org/EIPS/eip-3529 for why we do refunds
	refundAmount := gasToRefund(evmStateDB.GetRefund(), gasUsed)
	gasRemaining += refundAmount
	gasUsed -= refundAmount

	evmResp = &evm.MsgEthereumTxResponse{
		GasUsed: gasUsed,
		VmError: vmError,
		Ret:     returnBz,
		Logs:    evm.NewLogsFromEth(evmStateDB.Logs()),
		Hash:    txHash.Hex(),
	}

	if gasRemaining > msg.GasLimit { // rare case of overflow
		evmResp.GasUsed = msg.GasLimit // cap the gas used to the original gas limit
		return evmResp, sdkioerrors.Wrapf(core.ErrGasUintOverflow, "ApplyEvmMsg: message gas limit (%d) < leftover gas (%d)", msg.GasLimit, gasRemaining)
	}

	// The dirty states in `StateDB` is either committed or discarded after return
	if commit {
		if err := evmStateDB.Commit(); err != nil {
			return evmResp, sdkioerrors.Wrapf(err, "ApplyEvmMsg: %s", evm.ErrStateDBCommit)
		}
		evmObj.StateDB.Finalise( /*deleteEmptyObjects*/ false)
	}

	return evmResp, nil
}

func ParseWeiAsMultipleOfMicronibi(weiInt *big.Int) (
	newWeiInt *uint256.Int, err error,
) {
	// if "weiValue" is nil, 0, or negative, early return
	cmpSign := weiInt.Cmp(big.NewInt(0))
	if weiInt == nil {
		return (*uint256.Int)(nil), nil
	} else if cmpSign == 0 {
		return uint256.NewInt(0), nil
	} else if cmpSign < 0 {
		return newWeiInt, fmt.Errorf("wei parsing error: negative wei value cannot be a uint256 (%s)", weiInt)
	}

	// err if weiInt is too small
	tenPow12 := new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
	if weiInt.Cmp(tenPow12) < 0 {
		return newWeiInt, fmt.Errorf(
			"wei parsing error: wei amount is too small (%s), cannot transfer less than 1 micronibi. 1 NIBI == 10^6 micronibi == 10^18 wei", weiInt)
	}

	// truncate to highest micronibi amount
	newWeiInt, overflowed := uint256.FromBig(
		evm.NativeToWei(evm.WeiToNative(weiInt)),
	)
	if overflowed {
		return newWeiInt, fmt.Errorf("wei parsing error: overflow occurred in conversion from big.Int to uint256.Int for wei value %s", weiInt)
	}
	return newWeiInt, nil
}

// CreateFunToken is a gRPC transaction message for creating fungible token
// ("FunToken") a mapping between a bank coin and ERC20 token.
//
// If the mapping is generated from an ERC20, this tx creates a bank coin to go
// with it, and if the mapping's generated from a coin, the EVM module
// deploys an ERC20 contract that for which it will be the owner.
//
// ## Mapping an ERC20 Token a Newly Generated Bank Coin
//
// When an ERC20 token is used to create a FunToken mapping and corresponding
// Bank Coin, it must produce valid "bank.Metadata"
//
// Constraints:
//   - The first argument of DenomUnits is required and the official base unit
//     onchain, meaning the denom must be equivalent to bank.Metadata.Base.
//   - Coin `bank.Metadata.Display` must be a denom of one of the
//     `bank.Metadata.DenomUnits`. It is taken by Cosmos-SDK clients like wallets
//     to be "bank.DenomUnit" client takes the exponent from to display wallet
//     balances.
//
// Decimals for an ERC20 are synonymous to "bank.DenomUnit.Exponent" in what
// they mean for external clients like wallets.
func (k *Keeper) CreateFunToken(
	goCtx context.Context, msg *evm.MsgCreateFunToken,
) (resp *evm.MsgCreateFunTokenResponse, err error) {
	err = msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	// Deduct fee upon registration.
	ctx := sdk.UnwrapSDKContext(goCtx)
	err = k.deductCreateFunTokenFee(ctx, msg)
	if err != nil {
		return nil, err
	}

	var funtoken *evm.FunToken
	emptyErc20 := msg.FromErc20 == nil || msg.FromErc20.Size() == 0
	switch {
	case !emptyErc20 && msg.FromBankDenom == "":
		funtoken, err = k.createFunTokenFromERC20(
			ctx,
			msg.FromErc20.Address,
			msg.AllowZeroDecimals,
		)
	case emptyErc20 && msg.FromBankDenom != "":
		funtoken, err = k.createFunTokenFromCoin(
			ctx,
			msg.FromBankDenom,
			msg.AllowZeroDecimals,
		)
	default:
		// Impossible to reach this case due to ValidateBasic
		err = fmt.Errorf(
			"either the \"from_erc20\" or \"from_bank_denom\" must be set (but not both)")
	}
	if err != nil {
		return nil, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventFunTokenCreated{
		Creator:              msg.Sender,
		BankDenom:            funtoken.BankDenom,
		Erc20ContractAddress: funtoken.Erc20Addr.String(),
		IsMadeFromCoin:       emptyErc20,
	})

	return &evm.MsgCreateFunTokenResponse{
		FuntokenMapping: *funtoken,
	}, err
}

func (k Keeper) deductCreateFunTokenFee(ctx sdk.Context, msg *evm.MsgCreateFunToken) error {
	fee := k.FeeForCreateFunToken(ctx)
	from := sdk.MustAccAddressFromBech32(msg.Sender) // validation in msg.ValidateBasic

	if err := k.Bank.SendCoinsFromAccountToModule(
		ctx, from, evm.ModuleName, fee); err != nil {
		return fmt.Errorf("unable to pay the \"create_fun_token_fee\": %w", err)
	}
	if err := k.Bank.BurnCoins(ctx, evm.ModuleName, fee); err != nil {
		return fmt.Errorf("failed to burn the \"create_fun_token_fee\" after payment: %w", err)
	}
	return nil
}

func (k Keeper) FeeForCreateFunToken(ctx sdk.Context) sdk.Coins {
	evmParams := k.GetParams(ctx)
	return sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, evmParams.CreateFuntokenFee))
}

// ConvertCoinToEvm Sends a coin with a valid "FunToken" mapping to the
// given recipient address ("to_eth_addr") in the corresponding ERC20
// representation.
//
// In Nibiru v2.7.0, one special case was added to this function to handle
// wrapped NIBI (WNIBI) conversions. How this works is, if the Bank Coin given is
// NIBI, then WNIBI is treated as the fungible token mapping for the NIBI tokens.
// This can only happen if WNIBI contract is well-defined (non-empty bytecode at
// the contract address).
//
// If WNIBI is not well-defined, this function falls back to using the "FunToken"
// mapping if one is present. That cannot happen on mainnet (Eth Chain ID 6900),
// however it can for local networks and testnets, so we mention it here for
// completeness.
func (k *Keeper) ConvertCoinToEvm(
	goCtx context.Context, msg *evm.MsgConvertCoinToEvm,
) (resp *evm.MsgConvertCoinToEvmResponse, err error) {
	err = msg.ValidateBasic()
	if err != nil {
		return
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	senderBech32 := sdk.MustAccAddressFromBech32(msg.Sender)

	if msg.BankCoin.Denom == appconst.BondDenom {
		return k.convertCoinToEvmForWNIBI(
			ctx, msg, senderBech32,
		)
	}

	funTokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, msg.BankCoin.Denom))
	if len(funTokens) == 0 {
		return nil, fmt.Errorf("funtoken for bank denom \"%s\" does not exist", msg.BankCoin.Denom)
	}
	if len(funTokens) > 1 {
		return nil, fmt.Errorf("multiple funtokens for bank denom \"%s\" found", msg.BankCoin.Denom)
	}

	fungibleTokenMapping := funTokens[0]

	if fungibleTokenMapping.IsMadeFromCoin {
		return k.convertCoinToEvmBornCoin(
			ctx, senderBech32, msg.ToEthAddr.Address, msg.BankCoin, fungibleTokenMapping,
		)
	} else {
		return k.convertCoinToEvmBornERC20(
			ctx, senderBech32, msg.ToEthAddr.Address, msg.BankCoin, fungibleTokenMapping,
		)
	}
}

// ConvertEvmToCoin Sends an ERC20 token with a valid "FunToken" mapping to the
// given recipient address as a bank coin.
func (k *Keeper) ConvertEvmToCoin(
	goCtx context.Context, msg *evm.MsgConvertEvmToCoin,
) (resp *evm.MsgConvertEvmToCoinResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddrs, erc20, amount, toAddrs, err := msg.Validate()
	if err != nil {
		return
	}

	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	}
	defer func() {
		k.Bank.StateDB = nil
	}()

	// If the erc20 is WNIBI, attempt to unwrap the WNIBI
	evmParams := k.GetParams(ctx)
	if erc20.Hex() == evmParams.CanonicalWnibi.Hex() {
		_, err = k.ConvertEvmToCoinForWNIBI(
			ctx, stateDB, erc20, senderAddrs, toAddrs.Bech32,
			amount,
			nil, /*evmObj*/
		)
	} else {
		// Find the FunToken mapping for this ERC20
		funTokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20.Address))
		if len(funTokens) != 1 {
			err = fmt.Errorf("no FunToken mapping exists for ERC20 \"%s\"", erc20.Hex())
			return
		}

		funtokenMapping := funTokens[0]
		amountBig := amount.BigInt()
		if funtokenMapping.IsMadeFromCoin {
			err = k.convertEvmToCoinForCoinOriginated(
				ctx, senderAddrs, toAddrs.Bech32, erc20.Address, amountBig, funtokenMapping.BankDenom, stateDB,
			)
		} else {
			err = k.convertEvmToCoinForERC20Originated(
				ctx, senderAddrs, toAddrs.Bech32, erc20.Address, amountBig, funtokenMapping.BankDenom, stateDB,
			)
		}
	}

	if err != nil {
		return
	}
	if err = stateDB.Commit(); err != nil {
		return nil, sdkioerrors.Wrap(err, evm.ErrStateDBCommit)
	}

	return &evm.MsgConvertEvmToCoinResponse{}, nil
}

// EmitEthereumTxEvents emits all types of EVM events applicable to a particular execution case
func (k *Keeper) EmitEthereumTxEvents(
	ctx sdk.Context,
	recipient *gethcommon.Address,
	txType uint8,
	msg core.Message,
	evmResp *evm.MsgEthereumTxResponse,
) error {
	// Typed event: eth.evm.v1.EventEthereumTx
	eventEthereumTx := &evm.EventEthereumTx{
		EthHash: evmResp.Hash,
		Index:   strconv.FormatUint(k.EvmState.BlockTxIndex.GetOr(ctx, 0), 10),
		GasUsed: strconv.FormatUint(evmResp.GasUsed, 10),
	}
	if len(ctx.TxBytes()) > 0 {
		eventEthereumTx.Hash = tmbytes.HexBytes(cmttypes.Tx(ctx.TxBytes()).Hash()).String()
	}
	if recipient != nil {
		eventEthereumTx.Recipient = recipient.Hex()
	}
	if evmResp.Failed() {
		eventEthereumTx.VmError = evmResp.VmError
	}
	err := ctx.EventManager().EmitTypedEvent(eventEthereumTx)
	if err != nil {
		return sdkioerrors.Wrap(err, "EmitEthereumTxEvents: failed to emit event ethereum tx")
	}

	// Untyped event: "message", used for tendermint subscription
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, evm.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.From.Hex()),
			sdk.NewAttribute(evm.MessageEventAttrTxType, fmt.Sprintf("%d", txType)),
		),
	)

	// Emit typed events
	if !evmResp.Failed() {
		if recipient == nil { // contract creation
			contractAddr := crypto.CreateAddress(msg.From, msg.Nonce)
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventContractDeployed{
				Sender:       msg.From.Hex(),
				ContractAddr: contractAddr.String(),
			})
		} else if len(msg.Data) > 0 { // contract executed
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventContractExecuted{
				Sender:       msg.From.Hex(),
				ContractAddr: msg.To.String(),
			})
		} else if msg.Value.Cmp(big.NewInt(0)) > 0 { // evm transfer
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventTransfer{
				Sender:    msg.From.Hex(),
				Recipient: msg.To.Hex(),
				Amount:    msg.Value.String(),
			})
		}
	}

	return nil
}

// updateBlockBloom updates transient block bloom filter
func (k *Keeper) updateBlockBloom(
	ctx sdk.Context,
	evmResp *evm.MsgEthereumTxResponse,
	logIndex uint64,
) {
	if len(evmResp.Logs) > 0 {
		logs := evm.LogsToEthereum(evmResp.Logs)
		k.EvmState.BlockBloom.Set(ctx, k.EvmState.CalcBloomFromLogs(ctx, logs).Bytes())
		k.EvmState.BlockLogSize.Set(ctx, logIndex+uint64(len(logs)))
	}
}
