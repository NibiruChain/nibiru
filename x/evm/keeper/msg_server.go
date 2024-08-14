// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/x/evm/embeds"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

var _ evm.MsgServer = &Keeper{}

func (k *Keeper) EthereumTx(
	goCtx context.Context, msg *evm.MsgEthereumTx,
) (resp *evm.MsgEthereumTxResponse, err error) {
	if err := msg.ValidateBasic(); err != nil {
		return resp, errors.Wrap(err, "EthereumTx validate basic failed")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	tx := msg.AsTransaction()

	resp, err = k.ApplyEvmTx(ctx, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply transaction")
	}

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyAmount, tx.Value().String()),
		// add event for ethereum transaction hash format
		sdk.NewAttribute(evm.AttributeKeyEthereumTxHash, resp.Hash),
		// add event for index of valid ethereum tx
		sdk.NewAttribute(evm.AttributeKeyTxIndex, strconv.FormatUint(k.EvmState.BlockTxIndex.GetOr(ctx, 0), 10)),
		// add event for eth tx gas used, we can't get it from cosmos tx result when it contains multiple eth tx msgs.
		sdk.NewAttribute(evm.AttributeKeyTxGasUsed, strconv.FormatUint(resp.GasUsed, 10)),
	}

	if len(ctx.TxBytes()) > 0 {
		// add event for tendermint transaction hash format
		hash := tmbytes.HexBytes(tmtypes.Tx(ctx.TxBytes()).Hash())
		attrs = append(attrs, sdk.NewAttribute(evm.AttributeKeyTxHash, hash.String()))
	}

	if to := tx.To(); to != nil {
		attrs = append(attrs, sdk.NewAttribute(evm.AttributeKeyRecipient, to.Hex()))
	}

	if resp.Failed() {
		attrs = append(attrs, sdk.NewAttribute(evm.AttributeKeyEthereumTxFailed, resp.VmError))
	}

	txLogAttrs := make([]sdk.Attribute, len(resp.Logs))
	for i, log := range resp.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode log")
		}
		txLogAttrs[i] = sdk.NewAttribute(evm.AttributeKeyTxLog, string(value))
	}

	// emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			evm.EventTypeEthereumTx,
			attrs...,
		),
		sdk.NewEvent(
			evm.EventTypeTxLog,
			txLogAttrs...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, evm.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.From),
			sdk.NewAttribute(evm.AttributeKeyTxType, fmt.Sprintf("%d", tx.Type())),
		),
	})

	return resp, nil
}

func (k *Keeper) ApplyEvmTx(
	ctx sdk.Context, tx *gethcore.Transaction,
) (*evm.MsgEthereumTxResponse, error) {
	evmConfig, err := k.GetEVMConfig(ctx, ctx.BlockHeader().ProposerAddress, k.EthChainID(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load evm config")
	}
	txConfig := k.TxConfig(ctx, tx.Hash())

	// get the signer according to the chain rules from the config and block height
	signer := gethcore.MakeSigner(evmConfig.ChainConfig, big.NewInt(ctx.BlockHeight()))
	msg, err := tx.AsMessage(signer, evmConfig.BaseFee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to return ethereum transaction as core message")
	}

	tmpCtx, commit := ctx.CacheContext()

	// pass true to commit the StateDB
	res, err := k.ApplyEvmMsg(tmpCtx, msg, nil, true, evmConfig, txConfig)
	if err != nil {
		// when a transaction contains multiple msg, as long as one of the msg fails
		// all gas will be deducted. so is not msg.Gas()
		k.ResetGasMeterAndConsumeGas(ctx, ctx.GasMeter().Limit())
		return nil, errors.Wrap(err, "failed to apply ethereum core message")
	}

	logs := evm.LogsToEthereum(res.Logs)

	cumulativeGasUsed := res.GasUsed
	if ctx.BlockGasMeter() != nil {
		limit := ctx.BlockGasMeter().Limit()
		cumulativeGasUsed += ctx.BlockGasMeter().GasConsumed()
		if cumulativeGasUsed > limit {
			cumulativeGasUsed = limit
		}
	}

	var contractAddr gethcommon.Address
	if msg.To() == nil {
		contractAddr = crypto.CreateAddress(msg.From(), msg.Nonce())
	}

	receipt := &gethcore.Receipt{
		Type:              tx.Type(),
		PostState:         nil, // TODO: intermediate state root
		CumulativeGasUsed: cumulativeGasUsed,
		Bloom:             k.EvmState.CalcBloomFromLogs(ctx, logs),
		Logs:              logs,
		TxHash:            txConfig.TxHash,
		ContractAddress:   contractAddr,
		GasUsed:           res.GasUsed,
		BlockHash:         txConfig.BlockHash,
		BlockNumber:       big.NewInt(ctx.BlockHeight()),
		TransactionIndex:  txConfig.TxIndex,
	}

	if !res.Failed() {
		receipt.Status = gethcore.ReceiptStatusSuccessful
		commit()
		ctx.EventManager().EmitEvents(tmpCtx.EventManager().Events())

		// Emit typed events
		if msg.To() == nil { // contract creation
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventContractDeployed{
				Sender:       msg.From().String(),
				ContractAddr: contractAddr.String(),
			})
		} else if len(msg.Data()) > 0 { // contract executed
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventContractExecuted{
				Sender:       msg.From().String(),
				ContractAddr: msg.To().String(),
			})
		} else if msg.Value().Cmp(big.NewInt(0)) > 0 { // evm transfer
			_ = ctx.EventManager().EmitTypedEvent(&evm.EventTransfer{
				Sender:    msg.From().String(),
				Recipient: msg.To().String(),
				Amount:    msg.Value().String(),
			})
		}
	}

	// refund gas in order to match the Ethereum gas consumption instead of the default SDK one.
	if err = k.RefundGas(ctx, msg, msg.Gas()-res.GasUsed, evmConfig.Params.EvmDenom); err != nil {
		return nil, errors.Wrapf(err, "failed to refund gas leftover gas to sender %s", msg.From())
	}

	if len(receipt.Logs) > 0 {
		// Update transient block bloom filter
		k.EvmState.BlockBloom.Set(ctx, receipt.Bloom.Bytes())
		blockLogSize := uint64(txConfig.LogIndex) + uint64(len(receipt.Logs))
		k.EvmState.BlockLogSize.Set(ctx, blockLogSize)
	}

	blockTxIdx := uint64(txConfig.TxIndex) + 1
	k.EvmState.BlockTxIndex.Set(ctx, blockTxIdx)

	totalGasUsed, err := k.AddToBlockGasUsed(ctx, res.GasUsed)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add transient gas used")
	}

	// reset the gas meter for current cosmos transaction
	k.ResetGasMeterAndConsumeGas(ctx, totalGasUsed)
	return res, nil
}

// ApplyEvmMsgWithEmptyTxConfig; Computes new state by applyig the EVM
// message to the given state. This function calls [Keeper.ApplyEvmMsg] with
// and empty`statedb.TxConfig`.
// See [Keeper.ApplyEvmMsg].
func (k *Keeper) ApplyEvmMsgWithEmptyTxConfig(
	ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool,
) (*evm.MsgEthereumTxResponse, error) {
	cfg, err := k.GetEVMConfig(ctx, ctx.BlockHeader().ProposerAddress, k.EthChainID(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load evm config")
	}

	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash()))
	return k.ApplyEvmMsg(ctx, msg, tracer, commit, cfg, txConfig)
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
		Coinbase:    evmConfig.CoinBase,
		GasLimit:    eth.BlockGasLimit(ctx),
		BlockNumber: big.NewInt(ctx.BlockHeight()),
		Time:        big.NewInt(ctx.BlockHeader().Time.Unix()),
		Difficulty:  big.NewInt(0), // unused. Only required in PoW context
		BaseFee:     evmConfig.BaseFee,
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
			// Case 1: The requested height matches the one from the context, so we can retrieve the header
			// hash directly from the context.
			// Note: The headerHash is only set at begin block, it will be nil in case of a query context
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
			// Case 2: if the chain is not the current height we need to retrieve the hash from the store for the
			// current chain epoch. This only applies if the current height is greater than the requested height.
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
//
// It should be a `vm.Tracer` object or nil, if pass `nil`, it'll create a default one based on keeper options.
//
// # Commit parameter
//
// If commit is true, the `StateDB` will be committed, otherwise discarded.
func (k *Keeper) ApplyEvmMsg(ctx sdk.Context,
	msg core.Message,
	tracer vm.EVMLogger,
	commit bool,
	evmConfig *statedb.EVMConfig,
	txConfig statedb.TxConfig,
) (*evm.MsgEthereumTxResponse, error) {
	var (
		ret   []byte // return bytes from evm execution
		vmErr error  // vm errors do not effect consensus and are therefore not assigned to err
	)

	stateDB := statedb.New(ctx, k, txConfig)
	evmObj := k.NewEVM(ctx, msg, evmConfig, tracer, stateDB)

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
		return nil, errors.Wrap(err, "intrinsic gas failed")
	}

	// Should check again even if it is checked on Ante Handler, because eth_call don't go through Ante Handler.
	if leftoverGas < intrinsicGas {
		// eth_estimateGas will check for this exact error
		return nil, errors.Wrap(core.ErrIntrinsicGas, "apply message")
	}
	leftoverGas -= intrinsicGas

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
		return nil, err
		// return nil, fmt.Errorf("cannot use \"value\" in wei that can't be converted to unibi. %s is not divisible by 10^12", msg.Value())
	}

	if contractCreation {
		// take over the nonce management from evm:
		// - reset sender's nonce to msg.Nonce() before calling evm.
		// - increase sender's nonce by one no matter the result.
		stateDB.SetNonce(sender.Address(), msg.Nonce())
		ret, _, leftoverGas, vmErr = evmObj.Create(sender, msg.Data(), leftoverGas, msgWei)
		stateDB.SetNonce(sender.Address(), msg.Nonce()+1)
	} else {
		ret, leftoverGas, vmErr = evmObj.Call(sender, *msg.To(), msg.Data(), leftoverGas, msgWei)
	}

	// After EIP-3529: refunds are capped to gasUsed / 5
	refundQuotient := params.RefundQuotientEIP3529

	// calculate gas refund
	if msg.Gas() < leftoverGas {
		return nil, errors.Wrap(evm.ErrGasOverflow, "apply message")
	}
	// refund gas
	temporaryGasUsed := msg.Gas() - leftoverGas
	refund := GasToRefund(stateDB.GetRefund(), temporaryGasUsed, refundQuotient)

	// update leftoverGas and temporaryGasUsed with refund amount
	leftoverGas += refund
	temporaryGasUsed -= refund

	// EVM execution error needs to be available for the JSON-RPC client
	var vmError string
	if vmErr != nil {
		vmError = vmErr.Error()
	}

	// The dirty states in `StateDB` is either committed or discarded after return
	if commit {
		if err := stateDB.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit stateDB: %w", err)
		}
	}

	gasLimit := math.LegacyNewDec(int64(msg.Gas()))
	minGasMultiplier := k.GetMinGasMultiplier(ctx)
	minimumGasUsed := gasLimit.Mul(minGasMultiplier)

	if !minimumGasUsed.TruncateInt().IsUint64() {
		return nil, errors.Wrapf(evm.ErrGasOverflow, "minimumGasUsed(%s) is not a uint64", minimumGasUsed.TruncateInt().String())
	}

	if msg.Gas() < leftoverGas {
		return nil, errors.Wrapf(evm.ErrGasOverflow, "message gas limit < leftover gas (%d < %d)", msg.Gas(), leftoverGas)
	}

	gasUsed := math.LegacyMaxDec(minimumGasUsed, math.LegacyNewDec(int64(temporaryGasUsed))).TruncateInt().Uint64()
	// reset leftoverGas, to be used by the tracer
	leftoverGas = msg.Gas() - gasUsed

	return &evm.MsgEthereumTxResponse{
		GasUsed: gasUsed,
		VmError: vmError,
		Ret:     ret,
		Logs:    evm.NewLogsFromEth(stateDB.Logs()),
		Hash:    txConfig.TxHash.Hex(),
	}, nil
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
	var funtoken evm.FunToken
	err = msg.ValidateBasic()
	if err != nil {
		return
	}

	// Deduct fee upon registration.
	ctx := sdk.UnwrapSDKContext(goCtx)
	err = k.deductCreateFunTokenFee(ctx, msg)
	if err != nil {
		return
	}

	emptyErc20 := msg.FromErc20 == nil || msg.FromErc20.Size() == 0
	switch {
	case !emptyErc20 && msg.FromBankDenom == "":
		funtoken, err = k.CreateFunTokenFromERC20(ctx, *msg.FromErc20)
	case emptyErc20 && msg.FromBankDenom != "":
		funtoken, err = k.CreateFunTokenFromCoin(ctx, msg.FromBankDenom)
	default:
		// Impossible to reach this case due to ValidateBasic
		err = fmt.Errorf(
			"either the \"from_erc20\" or \"from_bank_denom\" must be set (but not both)")
	}
	if err != nil {
		return
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventFunTokenCreated{
		Creator:              msg.Sender,
		BankDenom:            funtoken.BankDenom,
		Erc20ContractAddress: funtoken.Erc20Addr.String(),
		IsMadeFromCoin:       emptyErc20,
	})

	return &evm.MsgCreateFunTokenResponse{
		FuntokenMapping: funtoken,
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
	return sdk.NewCoins(sdk.NewCoin(evmParams.EvmDenom, evmParams.CreateFuntokenFee))
}

// SendFunTokenToEvm Sends a coin with a valid "FunToken" mapping to the
// given recipient address ("to_eth_addr") in the corresponding ERC20
// representation.
func (k *Keeper) SendFunTokenToEvm(
	goCtx context.Context, msg *evm.MsgSendFunTokenToEvm,
) (resp *evm.MsgSendFunTokenToEvmResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	toEthAddr := msg.ToEthAddr.ToAddr()
	bankDenom := msg.BankCoin.Denom
	amount := msg.BankCoin.Amount

	funTokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom))
	if len(funTokens) == 0 {
		return nil, fmt.Errorf("funtoken for bank denom \"%s\" does not exist", bankDenom)
	}
	erc20ContractAddr := funTokens[0].Erc20Addr.ToAddr()

	// Step 1: Send coins to the evm module account
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, evm.ModuleName, sdk.Coins{msg.BankCoin})
	if err != nil {
		return nil, errors.Wrap(err, "failed to send coins to module account")
	}

	// Step 2: evm call to erc20 minter: mint tokens for a toEthAddr
	evmResp, err := k.CallContract(
		ctx,
		embeds.SmartContract_ERC20Minter.ABI,
		evm.EVM_MODULE_ADDRESS,
		&erc20ContractAddr,
		true,
		"mint",
		toEthAddr,
		amount.BigInt(),
	)
	if err != nil {
		return nil, err
	}
	if evmResp.Failed() {
		return nil,
			fmt.Errorf("failed to mint erc-20 tokens of contract %s", erc20ContractAddr.String())
	}
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventSendFunTokenToEvm{
		Sender:               msg.Sender,
		Erc20ContractAddress: erc20ContractAddr.String(),
		ToEthAddr:            toEthAddr.String(),
		BankCoin:             msg.BankCoin,
	})

	return &evm.MsgSendFunTokenToEvmResponse{}, nil
}
