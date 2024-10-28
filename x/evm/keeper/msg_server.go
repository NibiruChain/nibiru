// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

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
		return evmResp, errors.Wrap(err, "EthereumTx validate basic failed")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	tx := txMsg.AsTransaction()

	evmConfig, err := k.GetEVMConfig(ctx, ctx.BlockHeader().ProposerAddress, k.EthChainID(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load evm config")
	}
	txConfig := k.TxConfig(ctx, tx.Hash())

	// get the signer according to the chain rules from the config and block height
	signer := gethcore.MakeSigner(evmConfig.ChainConfig, big.NewInt(ctx.BlockHeight()))
	evmMsg, err := tx.AsMessage(signer, evmConfig.BaseFeeWei)
	if err != nil {
		return nil, errors.Wrap(err, "failed to return ethereum transaction as core message")
	}

	tmpCtx, commitCtx := ctx.CacheContext()

	// pass true to commit the StateDB
	evmResp, _, err = k.ApplyEvmMsg(tmpCtx, evmMsg, nil, true, evmConfig, txConfig, false)
	if err != nil {
		// when a transaction contains multiple msg, as long as one of the msg fails
		// all gas will be deducted. so is not msg.Gas()
		k.ResetGasMeterAndConsumeGas(ctx, ctx.GasMeter().Limit())
		return nil, errors.Wrap(err, "failed to apply ethereum core message")
	}

	if !evmResp.Failed() {
		commitCtx()
	}

	// refund gas in order to match the Ethereum gas consumption instead of the
	// default SDK one.
	refundGas := uint64(0)
	if evmMsg.Gas() > evmResp.GasUsed {
		refundGas = evmMsg.Gas() - evmResp.GasUsed
	}
	weiPerGas := txMsg.EffectiveGasPriceWeiPerGas(evmConfig.BaseFeeWei)
	if err = k.RefundGas(ctx, evmMsg.From(), refundGas, weiPerGas); err != nil {
		return nil, errors.Wrapf(err, "error refunding leftover gas to sender %s", evmMsg.From())
	}

	k.updateBlockBloom(ctx, evmResp, uint64(txConfig.LogIndex))

	totalGasUsed, err := k.AddToBlockGasUsed(ctx, evmResp.GasUsed)
	if err != nil {
		return nil, errors.Wrap(err, "error adding transient gas used to block")
	}

	// reset the gas meter for current TxMsg (EthereumTx)
	k.ResetGasMeterAndConsumeGas(ctx, totalGasUsed)

	err = k.EmitEthereumTxEvents(ctx, tx.To(), tx.Type(), evmMsg, evmResp)
	if err != nil {
		return nil, errors.Wrap(err, "error emitting ethereum tx events")
	}

	blockTxIdx := uint64(txConfig.TxIndex) + 1
	k.EvmState.BlockTxIndex.Set(ctx, blockTxIdx)
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
	evmConfig *statedb.EVMConfig,
	tracer vm.EVMLogger,
	stateDB vm.StateDB,
) *vm.EVM {
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     k.GetHashFn(ctx),
		Coinbase:    evmConfig.BlockCoinbase,
		GasLimit:    eth.BlockGasLimit(ctx),
		BlockNumber: big.NewInt(ctx.BlockHeight()),
		Time:        big.NewInt(ctx.BlockHeader().Time.Unix()),
		Difficulty:  big.NewInt(0), // unused. Only required in PoW context
		BaseFee:     evmConfig.BaseFeeWei,
		Random:      nil, // not supported
	}

	txCtx := core.NewEVMTxContext(msg)
	if tracer == nil {
		tracer = k.Tracer(ctx, msg, evmConfig.ChainConfig)
	}
	vmConfig := k.VMConfig(ctx, msg, evmConfig, tracer)
	theEvm := vm.NewEVM(blockCtx, txCtx, stateDB, evmConfig.ChainConfig, vmConfig)
	theEvm.WithPrecompiles(k.precompiles.InternalData(), k.precompiles.Keys())
	return theEvm
}

// GetHashFn implements vm.GetHashFunc for Ethermint. It handles 3 cases:
//  1. The requested height matches the current height from context (and thus same epoch number)
//  2. The requested height is from a previous height from the same chain epoch
//  3. The requested height is from a height greater than the latest one
func (k Keeper) GetHashFn(ctx sdk.Context) vm.GetHashFunc {
	return func(height uint64) gethcommon.Hash {
		h, err := eth.SafeInt64(height)
		if err != nil {
			k.Logger(ctx).Error("failed to cast height to int64", "error", err)
			return gethcommon.Hash{}
		}

		switch {
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
			header, err := tmtypes.HeaderFromProto(&contextBlockHeader)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err)
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

			header, err := tmtypes.HeaderFromProto(&histInfo.Header)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err)
				return gethcommon.Hash{}
			}

			return gethcommon.BytesToHash(header.Hash())
		default:
			// Case 3: heights greater than the current one returns an empty hash.
			return gethcommon.Hash{}
		}
	}
}

// ApplyEvmMsg computes the new state by applying the given message against the existing state.
// If the message fails, the VM execution error with the reason will be returned to the client
// and the transaction won't be committed to the store.
//
// # Reverted state
//
// The snapshot and rollback are supported by the `statedb.StateDB`.
//
// # Different Callers
//
// It's called in three scenarios:
// 1. `ApplyTransaction`, in the transaction processing flow.
// 2. `EthCall/EthEstimateGas` grpc query handler.
// 3. Called by other native modules directly.
//
// # Prechecks and Preprocessing
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
// # Tracer parameter
// It should be a `vm.Tracer` object or nil, if pass `nil`, it'll create a default one based on keeper options.
//
// # Commit parameter
// If commit is true, the `StateDB` will be committed, otherwise discarded.
//
// # fullRefundLeftoverGas parameter
// For internal calls like funtokens, user does not specify gas limit explicitly.
// In this case we don't apply any caps for refund and refund 100%
func (k *Keeper) ApplyEvmMsg(ctx sdk.Context,
	msg core.Message,
	tracer vm.EVMLogger,
	commit bool,
	evmConfig *statedb.EVMConfig,
	txConfig statedb.TxConfig,
	fullRefundLeftoverGas bool,
) (resp *evm.MsgEthereumTxResponse, evmObj *vm.EVM, err error) {
	var (
		ret   []byte // return bytes from evm execution
		vmErr error  // vm errors do not effect consensus and are therefore not assigned to err
	)

	stateDB := statedb.New(ctx, k, txConfig)
	evmObj = k.NewEVM(ctx, msg, evmConfig, tracer, stateDB)

	leftoverGas := msg.Gas()

	// Allow the tracer captures the tx level events, mainly the gas consumption.
	vmCfg := evmObj.Config
	if vmCfg.Debug {
		vmCfg.Tracer.CaptureTxStart(leftoverGas)
		defer func() {
			vmCfg.Tracer.CaptureTxEnd(leftoverGas)
		}()
	}

	sender := vm.AccountRef(msg.From())
	contractCreation := msg.To() == nil
	intrinsicGas, err := k.GetEthIntrinsicGas(ctx, msg, evmConfig.ChainConfig, contractCreation)
	if err != nil {
		// should have already been checked on Ante Handler
		return nil, evmObj, errors.Wrap(err, "intrinsic gas failed")
	}

	// Check if the provided gas in the message is enough to cover the intrinsic
	// gas, the base gas cost before execution occurs (gethparams.TxGas, contract
	// creation, and cost per byte of the data payload).
	//
	// Should check again even if it is checked on Ante Handler, because eth_call
	// don't go through Ante Handler.
	if leftoverGas < intrinsicGas {
		// eth_estimateGas will check for this exact error
		return nil, evmObj, errors.Wrapf(
			core.ErrIntrinsicGas,
			"apply message msg.Gas = %d, intrinsic gas = %d.",
			leftoverGas, intrinsicGas,
		)
	}
	leftoverGas = leftoverGas - intrinsicGas

	// access list preparation is moved from ante handler to here, because it's
	// needed when `ApplyMessage` is called under contexts where ante handlers
	// are not run, for example `eth_call` and `eth_estimateGas`.
	stateDB.PrepareAccessList(
		msg.From(),
		msg.To(),
		evmObj.ActivePrecompiles(params.Rules{}),
		msg.AccessList(),
	)

	msgWei, err := ParseWeiAsMultipleOfMicronibi(msg.Value())
	if err != nil {
		return nil, evmObj, err
	}

	if contractCreation {
		// take over the nonce management from evm:
		// - reset sender's nonce to msg.Nonce() before calling evm.
		// - increase sender's nonce by one no matter the result.
		stateDB.SetNonce(sender.Address(), msg.Nonce())
		ret, _, leftoverGas, vmErr = evmObj.Create(
			sender,
			msg.Data(),
			leftoverGas,
			msgWei,
		)
		stateDB.SetNonce(sender.Address(), msg.Nonce()+1)
	} else {
		ret, leftoverGas, vmErr = evmObj.Call(
			sender,
			*msg.To(),
			msg.Data(),
			leftoverGas,
			msgWei,
		)
	}

	// EVM execution error needs to be available for the JSON-RPC client
	var vmError string
	if vmErr != nil {
		vmError = vmErr.Error()
	}

	// The dirty states in `StateDB` is either committed or discarded after return
	if commit {
		if err := stateDB.Commit(); err != nil {
			return nil, evmObj, fmt.Errorf("failed to commit stateDB: %w", err)
		}
	}
	// Rare case of uint64 gas overflow
	if msg.Gas() < leftoverGas {
		return nil, evmObj, errors.Wrap(evm.ErrGasOverflow, "apply message")
	}

	// GAS REFUND
	// If msg.Gas() > gasUsed, we need to refund extra gas.
	// leftoverGas = amount of extra (not used) gas.
	// If the msg comes from user, we apply refundQuotient capping the refund to 20% of used gas
	// If msg is internal (funtoken), we refund 100%

	refundQuotient := params.RefundQuotientEIP3529  // EIP-3529: refunds are capped to gasUsed / 5
	minGasUsedPct := k.GetMinGasUsedMultiplier(ctx) // Evmos invention: https://github.com/evmos/ethermint/issues/1085
	if fullRefundLeftoverGas {
		refundQuotient = 1                   // 100% refund
		minGasUsedPct = math.LegacyZeroDec() // no minimum, get the actual gasUsed value
	}
	temporaryGasUsed := msg.Gas() - leftoverGas
	refund := GasToRefund(stateDB.GetRefund(), temporaryGasUsed, refundQuotient)

	// update leftoverGas and temporaryGasUsed with refund amount
	leftoverGas += refund
	temporaryGasUsed -= refund
	if msg.Gas() < leftoverGas {
		return nil, evmObj, errors.Wrapf(evm.ErrGasOverflow, "message gas limit < leftover gas (%d < %d)", msg.Gas(), leftoverGas)
	}

	// Min gas used is a % of gasLimit
	minimumGasUsed := math.LegacyNewDec(int64(msg.Gas())).Mul(minGasUsedPct)
	gasUsed := math.LegacyMaxDec(minimumGasUsed, math.LegacyNewDec(int64(temporaryGasUsed))).TruncateInt().Uint64()

	// This resulting "leftoverGas" is used by the tracer. This happens as a
	// result of the defer statement near the beginning of the function with
	// "vm.Tracer".
	leftoverGas = msg.Gas() - gasUsed

	return &evm.MsgEthereumTxResponse{
		GasUsed: gasUsed,
		VmError: vmError,
		Ret:     ret,
		Logs:    evm.NewLogsFromEth(stateDB.Logs()),
		Hash:    txConfig.TxHash.Hex(),
	}, evmObj, nil
}

func ParseWeiAsMultipleOfMicronibi(weiInt *big.Int) (newWeiInt *big.Int, err error) {
	// if "weiValue" is nil, 0, or negative, early return
	if weiInt == nil || !(weiInt.Cmp(big.NewInt(0)) > 0) {
		return weiInt, nil
	}

	// err if weiInt is too small
	tenPow12 := new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
	if weiInt.Cmp(tenPow12) < 0 {
		return weiInt, fmt.Errorf(
			"wei amount is too small (%s), cannot transfer less than 1 micronibi. 10^18 wei == 1 NIBI == 10^6 micronibi", weiInt)
	}

	// truncate to highest micronibi amount
	newWeiInt = evm.NativeToWei(evm.WeiToNative(weiInt))
	return newWeiInt, nil
}

// CreateFunToken is a gRPC transaction message for creating fungible token
// ("FunToken") a mapping between a bank coin and ERC20 token.
//
// If the mapping is generated from an ERC20, this tx creates a bank coin to go
// with it, and if the mapping's generated from a coin, the EVM module
// deploys an ERC20 contract that for which it will be the owner.
func (k *Keeper) CreateFunToken(
	goCtx context.Context, msg *evm.MsgCreateFunToken,
) (resp *evm.MsgCreateFunTokenResponse, err error) {
	var funtoken *evm.FunToken
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

	emptyErc20 := msg.FromErc20 == nil || msg.FromErc20.Size() == 0
	switch {
	case !emptyErc20 && msg.FromBankDenom == "":
		funtoken, err = k.createFunTokenFromERC20(ctx, msg.FromErc20.Address)
	case emptyErc20 && msg.FromBankDenom != "":
		funtoken, err = k.createFunTokenFromCoin(ctx, msg.FromBankDenom)
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

	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, from, evm.ModuleName, fee); err != nil {
		return fmt.Errorf("unable to pay the \"create_fun_token_fee\": %w", err)
	}
	if err := k.bankKeeper.BurnCoins(ctx, evm.ModuleName, fee); err != nil {
		return fmt.Errorf("failed to burn the \"create_fun_token_fee\" after payment: %w", err)
	}
	return nil
}

func (k Keeper) FeeForCreateFunToken(ctx sdk.Context) sdk.Coins {
	evmParams := k.GetParams(ctx)
	return sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, evmParams.CreateFuntokenFee))
}

// EmitEthereumTxEvents emits all types of EVM events applicable to a particular execution case
func (k *Keeper) EmitEthereumTxEvents(
	ctx sdk.Context,
	recipient *gethcommon.Address,
	txType uint8,
	msg gethcore.Message,
	evmResp *evm.MsgEthereumTxResponse,
) error {
	// Typed event: eth.evm.v1.EventEthereumTx
	eventEthereumTx := &evm.EventEthereumTx{
		EthHash: evmResp.Hash,
		Index:   strconv.FormatUint(k.EvmState.BlockTxIndex.GetOr(ctx, 0), 10),
		GasUsed: strconv.FormatUint(evmResp.GasUsed, 10),
	}
	if len(ctx.TxBytes()) > 0 {
		eventEthereumTx.Hash = tmbytes.HexBytes(tmtypes.Tx(ctx.TxBytes()).Hash()).String()
	}
	if recipient != nil {
		eventEthereumTx.Recipient = recipient.Hex()
	}
	if evmResp.Failed() {
		eventEthereumTx.EthTxFailed = evmResp.VmError
	}
	err := ctx.EventManager().EmitTypedEvent(eventEthereumTx)
	if err != nil {
		return errors.Wrap(err, "failed to emit event ethereum tx")
	}

	// Typed event: eth.evm.v1.EventTxLog
	txLogs := make([]string, len(evmResp.Logs))
	for i, log := range evmResp.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return errors.Wrap(err, "failed to encode log")
		}
		txLogs[i] = string(value)
	}
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{TxLogs: txLogs})

	// Untyped event: "message", used for tendermint subscription
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, evm.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.From().Hex()),
			sdk.NewAttribute(evm.MessageEventAttrTxType, fmt.Sprintf("%d", txType)),
		),
	)

	// Emit typed events
	if !evmResp.Failed() {
		if recipient == nil { // contract creation
			contractAddr := crypto.CreateAddress(msg.From(), msg.Nonce())
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventContractDeployed{
				Sender:       msg.From().Hex(),
				ContractAddr: contractAddr.String(),
			})
		} else if len(msg.Data()) > 0 { // contract executed
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventContractExecuted{
				Sender:       msg.From().Hex(),
				ContractAddr: msg.To().String(),
			})
		} else if msg.Value().Cmp(big.NewInt(0)) > 0 { // evm transfer
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventTransfer{
				Sender:    msg.From().Hex(),
				Recipient: msg.To().Hex(),
				Amount:    msg.Value().String(),
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
	logs := evm.LogsToEthereum(evmResp.Logs)
	if len(logs) > 0 {
		k.EvmState.BlockBloom.Set(ctx, k.EvmState.CalcBloomFromLogs(ctx, logs).Bytes())
		blockLogSize := logIndex + uint64(len(logs))
		k.EvmState.BlockLogSize.Set(ctx, blockLogSize)
	}
}

// CallContract invokes a smart contract on the method specified by [methodName]
// using the given [args].
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - abi: The ABI (Application Binary Interface) of the smart contract.
//   - fromAcc: The Ethereum address of the account initiating the contract call.
//   - contract: Pointer to the Ethereum address of the contract to be called.
//   - commit: Boolean flag indicating whether to commit the transaction (true) or simulate it (false).
//   - methodName: The name of the contract method to be called.
//   - args: Variadic parameter for the arguments to be passed to the contract method.
//
// Note: This function handles both contract method calls and simulations,
// depending on the 'commit' parameter.
func (k Keeper) CallContract(
	ctx sdk.Context,
	abi *gethabi.ABI,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	gasLimit uint64,
	methodName string,
	args ...any,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	contractInput, err := abi.Pack(methodName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI args: %w", err)
	}
	evmResp, _, err = k.CallContractWithInput(ctx, fromAcc, contract, commit, contractInput, gasLimit)
	return evmResp, err
}

// CallContractWithInput invokes a smart contract with the given [contractInput]
// or deploys a new contract.
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - fromAcc: The Ethereum address of the account initiating the contract call.
//   - contract: Pointer to the Ethereum address of the contract. Nil if new
//     contract is deployed.
//   - commit: Boolean flag indicating whether to commit the transaction (true)
//     or simulate it (false).
//   - contractInput: Hexadecimal-encoded bytes use as input data to the call.
//
// Note: This function handles both contract method calls and simulations,
// depending on the 'commit' parameter. It uses a default gas limit.
func (k Keeper) CallContractWithInput(
	ctx sdk.Context,
	fromAcc gethcommon.Address,
	contract *gethcommon.Address,
	commit bool,
	contractInput []byte,
	gasLimit uint64,
) (evmResp *evm.MsgEthereumTxResponse, evmObj *vm.EVM, err error) {
	// This is a `defer` pattern to add behavior that runs in the case that the
	// error is non-nil, creating a concise way to add extra information.
	defer func() {
		if err != nil {
			err = fmt.Errorf("CallContractError: %w", err)
		}
	}()
	nonce := k.GetAccNonce(ctx, fromAcc)

	unusedBigInt := big.NewInt(0)
	evmMsg := gethcore.NewMessage(
		fromAcc,
		contract,
		nonce,
		unusedBigInt, // amount
		gasLimit,
		unusedBigInt, // gasFeeCap
		unusedBigInt, // gasTipCap
		unusedBigInt, // gasPrice
		contractInput,
		gethcore.AccessList{},
		!commit, // isFake
	)

	// Apply EVM message
	evmCfg, err := k.GetEVMConfig(
		ctx,
		sdk.ConsAddress(ctx.BlockHeader().ProposerAddress),
		k.EthChainID(ctx),
	)
	if err != nil {
		err = errors.Wrapf(err, "failed to load EVM config")
		return
	}

	// Generating TxConfig with an empty tx hash as there is no actual eth tx
	// sent by a user
	txConfig := k.TxConfig(ctx, gethcommon.BigToHash(big.NewInt(0)))

	// Using tmp context to not modify the state in case of evm revert
	tmpCtx, commitCtx := ctx.CacheContext()

	evmResp, evmObj, err = k.ApplyEvmMsg(
		tmpCtx, evmMsg, evm.NewNoOpTracer(), commit, evmCfg, txConfig, true,
	)
	if err != nil {
		// We don't know the actual gas used, so consuming the gas limit
		k.ResetGasMeterAndConsumeGas(ctx, gasLimit)
		err = errors.Wrap(err, "failed to apply ethereum core message")
		return
	}
	if evmResp.Failed() {
		k.ResetGasMeterAndConsumeGas(ctx, evmResp.GasUsed)
		if !strings.Contains(evmResp.VmError, vm.ErrOutOfGas.Error()) {
			if evmResp.VmError == vm.ErrExecutionReverted.Error() {
				err = fmt.Errorf("VMError: %w", evm.NewExecErrorWithReason(evmResp.Ret))
				return
			}
			err = fmt.Errorf("VMError: %s", evmResp.VmError)
			return
		}
		err = fmt.Errorf("gas required exceeds allowance (%d)", gasLimit)
		return
	} else {
		// Success, committing the state to ctx
		if commit {
			commitCtx()
			totalGasUsed, err := k.AddToBlockGasUsed(ctx, evmResp.GasUsed)
			if err != nil {
				k.ResetGasMeterAndConsumeGas(ctx, ctx.GasMeter().Limit())
				return nil, nil, errors.Wrap(err, "error adding transient gas used to block")
			}
			k.ResetGasMeterAndConsumeGas(ctx, totalGasUsed)
			k.updateBlockBloom(ctx, evmResp, uint64(txConfig.LogIndex))
			err = k.EmitEthereumTxEvents(ctx, contract, gethcore.LegacyTxType, evmMsg, evmResp)
			if err != nil {
				return nil, nil, errors.Wrap(err, "error emitting ethereum tx events")
			}
			blockTxIdx := uint64(txConfig.TxIndex) + 1
			k.EvmState.BlockTxIndex.Set(ctx, blockTxIdx)
		}
		return evmResp, evmObj, nil
	}
}
