// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	sdkmath "cosmossdk.io/math"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/set"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	gethparams "github.com/ethereum/go-ethereum/params"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

// Compile-time interface assertion
var _ evm.QueryServer = &Keeper{}

// EthAccount: Implements the gRPC query for "/eth.evm.v1.Query/EthAccount".
// EthAccount retrieves the account  and balance details for an account with the
// given address.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: Request containing the address in either Ethereum hexadecimal or
//     Bech32 format.
func (k Keeper) EthAccount(
	goCtx context.Context, req *evm.QueryEthAccountRequest,
) (*evm.QueryEthAccountResponse, error) {
	isBech32, err := req.Validate()
	if err != nil {
		return nil, err
	}

	var addrEth gethcommon.Address
	var addrBech32 sdk.AccAddress

	if isBech32 {
		addrBech32 = sdk.MustAccAddressFromBech32(req.Address)
		addrEth = eth.NibiruAddrToEthAddr(addrBech32)
	} else {
		addrEth = gethcommon.HexToAddress(req.Address)
		addrBech32 = eth.EthAddrToNibiruAddr(addrEth)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	acct := k.getAccountWithoutBalance(ctx, addrEth)
	if acct == nil {
		return nil, fmt.Errorf("account not found for %s", addrEth.Hex())
	}
	balNative := k.Bank.GetBalance(ctx, addrBech32, evm.EVMBankDenom).Amount.BigInt()

	return &evm.QueryEthAccountResponse{
		EthAddress:    addrEth.Hex(),
		Bech32Address: addrBech32.String(),
		Balance:       balNative.String(),
		BalanceWei:    evm.NativeToWei(balNative).String(),
		CodeHash:      gethcommon.BytesToHash(acct.CodeHash).Hex(),
		Nonce:         acct.Nonce,
	}, nil
}

// ValidatorAccount: Implements the gRPC query for
// "/eth.evm.v1.Query/ValidatorAccount". ValidatorAccount retrieves the account
// details for a given validator consensus address.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: Request containing the validator consensus address.
//
// Returns:
//   - Response containing the account details.
//   - An error if the account retrieval process encounters any issues.
func (k Keeper) ValidatorAccount(
	goCtx context.Context, req *evm.QueryValidatorAccountRequest,
) (*evm.QueryValidatorAccountResponse, error) {
	consAddr, err := req.Validate()
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	validator, found := k.stakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if !found {
		return nil, fmt.Errorf("validator not found for %s", consAddr.String())
	}

	nibiruAddr := sdk.AccAddress(validator.GetOperator())
	res := evm.QueryValidatorAccountResponse{
		AccountAddress: nibiruAddr.String(),
	}

	account := k.accountKeeper.GetAccount(ctx, nibiruAddr)
	if account != nil {
		res.Sequence = account.GetSequence()
		res.AccountNumber = account.GetAccountNumber()
	}

	return &res, nil
}

// Balance: Implements the gRPC query for "/eth.evm.v1.Query/Balance".
// Balance retrieves the balance of an Ethereum address in "wei", the smallest
// unit of "Ether". Ether refers to NIBI tokens on Nibiru EVM.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: The QueryBalanceRequest object containing the Ethereum address.
//
// Returns:
//   - A pointer to the QueryBalanceResponse object containing the balance.
//   - An error if the balance retrieval process encounters any issues.
func (k Keeper) Balance(goCtx context.Context, req *evm.QueryBalanceRequest) (*evm.QueryBalanceResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	balanceInt := k.GetEvmGasBalance(ctx, gethcommon.HexToAddress(req.Address))
	return &evm.QueryBalanceResponse{
		Balance:    balanceInt.String(),
		BalanceWei: evm.NativeToWei(balanceInt).String(),
	}, nil
}

// BaseFee implements the Query/BaseFee gRPC method
func (k Keeper) BaseFee(
	goCtx context.Context, _ *evm.QueryBaseFeeRequest,
) (*evm.QueryBaseFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	baseFeeMicronibiPerGas := sdkmath.NewIntFromBigInt(k.BaseFeeMicronibiPerGas(ctx))
	baseFeeWei := sdkmath.NewIntFromBigInt(
		evm.NativeToWei(baseFeeMicronibiPerGas.BigInt()),
	)
	return &evm.QueryBaseFeeResponse{
		BaseFee:      &baseFeeWei,
		BaseFeeUnibi: &baseFeeMicronibiPerGas,
	}, nil
}

// Storage: Implements the gRPC query for "/eth.evm.v1.Query/Storage".
// Storage retrieves the storage value for a given Ethereum address and key.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: The QueryStorageRequest object containing the Ethereum address and key.
//
// Returns:
//   - A pointer to the QueryStorageResponse object containing the storage value.
//   - An error if the storage retrieval process encounters any issues.
func (k Keeper) Storage(
	goCtx context.Context, req *evm.QueryStorageRequest,
) (*evm.QueryStorageResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	address := gethcommon.HexToAddress(req.Address)
	key := gethcommon.HexToHash(req.Key)

	state := k.GetState(ctx, address, key)
	stateHex := state.Hex()

	return &evm.QueryStorageResponse{
		Value: stateHex,
	}, nil
}

// Code: Implements the gRPC query for "/eth.evm.v1.Query/Code".
// Code retrieves the smart contract bytecode associated with a given Ethereum
// address.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: Request with the Ethereum address of the smart contract bytecode.
//
// Returns:
//   - Response containing the smart contract bytecode.
//   - An error if the code retrieval process encounters any issues.
func (k Keeper) Code(
	goCtx context.Context, req *evm.QueryCodeRequest,
) (*evm.QueryCodeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	address := gethcommon.HexToAddress(req.Address)
	acct := k.getAccountWithoutBalance(ctx, address)

	var code []byte
	if acct != nil && acct.IsContract() {
		code = k.GetCode(ctx, gethcommon.BytesToHash(acct.CodeHash))
	}

	return &evm.QueryCodeResponse{
		Code: code,
	}, nil
}

// Params: Implements the gRPC query for "/eth.evm.v1.Query/Params".
// Params retrieves the EVM module parameters.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: The QueryParamsRequest object (unused).
//
// Returns:
//   - A pointer to the QueryParamsResponse object containing the EVM module parameters.
//   - An error if the parameter retrieval process encounters any issues.
func (k Keeper) Params(
	goCtx context.Context, _ *evm.QueryParamsRequest,
) (*evm.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &evm.QueryParamsResponse{
		Params: params,
	}, nil
}

// EthCall: Implements the gRPC query for "/eth.evm.v1.Query/EthCall".
// EthCall performs a smart contract call using the eth_call JSON-RPC method.
//
// An "eth_call" is a method from the Ethereum JSON-RPC specification that allows
// one to call a smart contract function without execution a transaction on the
// blockchain. This is useful for simulating transactions and for reading data
// from the chain using responses from smart contract calls.
//
// Parameters:
//   - goCtx: Request context with information about the current block that
//     serves as the main access point to the blockchain state.
//   - req: "eth_call" parameters to interact with a smart contract.
//
// Returns:
//   - A pointer to the MsgEthereumTxResponse object containing the result of the eth_call.
//   - An error if the eth_call process encounters any issues.
func (k *Keeper) EthCall(
	goCtx context.Context, req *evm.EthCallRequest,
) (*evm.MsgEthereumTxResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var args evm.JsonTxArgs
	err := json.Unmarshal(req.Args, &args)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, err.Error())
	}
	evmCfg := k.GetEVMConfig(ctx)

	// ApplyMessageWithConfig expect correct nonce set in msg
	nonce := k.GetAccNonce(ctx, args.GetFrom())
	args.Nonce = (*hexutil.Uint64)(&nonce)

	msg, err := args.ToMessage(req.GasCap, evmCfg.BaseFeeWei)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, err.Error())
	}

	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash()))

	// pass false to not commit StateDB
	stateDB := statedb.New(ctx, k, txConfig)
	evm := k.NewEVM(ctx, msg, evmCfg, nil /*tracer*/, stateDB)
	res, err := k.ApplyEvmMsg(ctx, msg, evm, false /*commit*/, txConfig.TxHash)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	return res, nil
}

// EstimateGas: Implements the gRPC query for "/eth.evm.v1.Query/EstimateGas".
// EstimateGas implements eth_estimateGas rpc api.
func (k Keeper) EstimateGas(
	goCtx context.Context, req *evm.EthCallRequest,
) (*evm.EstimateGasResponse, error) {
	return k.EstimateGasForEvmCallType(goCtx, req, evm.CallTypeRPC)
}

// EstimateGasForEvmCallType estimates the gas cost of a transaction. This can be
// called with the "eth_estimateGas" JSON-RPC method or smart contract query.
//
// When [EstimateGas] is called from the JSON-RPC client, we need to reset the
// gas meter before simulating the transaction (tx) to have an accurate gas
// estimate txs using EVM extensions.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: The EthCallRequest object containing the transaction parameters.
//
// Returns:
//   - A response containing the estimated gas cost.
//   - An error if the gas estimation process encounters any issues.
func (k Keeper) EstimateGasForEvmCallType(
	goCtx context.Context, req *evm.EthCallRequest, fromType evm.CallType,
) (*evm.EstimateGasResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	evmCfg := k.GetEVMConfig(ctx)

	if req.GasCap < gethparams.TxGas {
		return nil, grpcstatus.Errorf(grpccodes.InvalidArgument, "gas cap cannot be lower than %d", gethparams.TxGas)
	}

	var args evm.JsonTxArgs
	err := json.Unmarshal(req.Args, &args)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, err.Error())
	}

	// ApplyMessageWithConfig expect correct nonce set in msg
	nonce := k.GetAccNonce(ctx, args.GetFrom())
	args.Nonce = (*hexutil.Uint64)(&nonce)

	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo     = gethparams.TxGas - 1
		hi     uint64
		gasCap uint64
	)

	// Determine the highest gas limit can be used during the estimation.
	if args.Gas != nil && uint64(*args.Gas) >= gethparams.TxGas {
		hi = uint64(*args.Gas)
	} else {
		// Query block gas limit
		params := ctx.ConsensusParams()
		if params != nil && params.Block != nil && params.Block.MaxGas > 0 {
			hi = uint64(params.Block.MaxGas)
		} else {
			hi = req.GasCap
		}
	}

	// TODO: Recap the highest gas limit with account's available balance.
	// Recap the highest gas allowance with specified gascap.
	if req.GasCap != 0 && hi > req.GasCap {
		hi = req.GasCap
	}

	gasCap = hi

	// convert the tx args to an ethereum message
	evmMsg, err := args.ToMessage(req.GasCap, evmCfg.BaseFeeWei)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	// NOTE: the errors from the executable below should be consistent with
	// go-ethereum, so we don't wrap them with the gRPC status code Create a
	// helper to check if a gas allowance results in an executable transaction.
	executable := func(gas uint64) (vmError bool, rsp *evm.MsgEthereumTxResponse, err error) {
		// update the message with the new gas value
		evmMsg = core.Message{
			To:               evmMsg.To,
			From:             evmMsg.From,
			Nonce:            evmMsg.Nonce,
			Value:            evmMsg.Value,
			GasLimit:         gas, // <---- This one changed
			GasPrice:         evmMsg.GasPrice,
			GasFeeCap:        evmMsg.GasFeeCap,
			GasTipCap:        evmMsg.GasTipCap,
			Data:             evmMsg.Data,
			AccessList:       evmMsg.AccessList,
			BlobGasFeeCap:    evmMsg.BlobGasFeeCap,
			BlobHashes:       evmMsg.BlobHashes,
			SkipNonceChecks:  evmMsg.SkipNonceChecks,
			SkipFromEOACheck: evmMsg.SkipFromEOACheck,
		}

		tmpCtx := ctx
		if fromType == evm.CallTypeRPC {
			tmpCtx, _ = ctx.CacheContext()

			acct := k.GetAccount(tmpCtx, evmMsg.From)

			from := evmMsg.From
			if acct == nil {
				acc := k.accountKeeper.NewAccountWithAddress(tmpCtx, from[:])
				k.accountKeeper.SetAccount(tmpCtx, acc)
				acct = statedb.NewEmptyAccount()
			}
			// When submitting a transaction, the `EthIncrementSenderSequence` ante handler increases the account nonce
			acct.Nonce = nonce + 1
			err = k.SetAccount(tmpCtx, from, *acct)
			if err != nil {
				return true, nil, err
			}
			// resetting the gasMeter after increasing the sequence to have an accurate gas estimation on EVM extensions transactions
			gasMeter := eth.NewInfiniteGasMeterWithLimit(evmMsg.GasLimit)
			tmpCtx = tmpCtx.WithGasMeter(gasMeter).
				WithKVGasConfig(storetypes.GasConfig{}).
				WithTransientKVGasConfig(storetypes.GasConfig{})
		}
		// pass false to not commit StateDB
		txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes()))
		stateDB := statedb.New(ctx, &k, txConfig)
		evmObj := k.NewEVM(tmpCtx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
		rsp, err = k.ApplyEvmMsg(tmpCtx, evmMsg, evmObj, false /*commit*/, txConfig.TxHash)
		if err != nil {
			if errors.Is(err, core.ErrIntrinsicGas) {
				return true, nil, nil // Special case, raise gas limit
			}
			return true, nil, fmt.Errorf("error applying EVM message to StateDB: %w", err) // Bail out
		}
		return len(rsp.VmError) > 0, rsp, nil
	}

	// Execute the binary search and hone in on an executable gas limit
	hi, err = evm.BinSearch(lo, hi, executable)
	if err != nil {
		return nil, err
	}

	// The gas limit is now the highest gas limit that results in an executable transaction
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == gasCap {
		failed, result, err := executable(hi)
		if err != nil {
			return nil, fmt.Errorf("eth call exec error: %w", err)
		}

		if failed && result != nil {
			if result.VmError == vm.ErrExecutionReverted.Error() {
				return nil, fmt.Errorf("estimate gas VMError: %w", evm.NewRevertError(result.Ret))
			}

			if result.VmError == vm.ErrOutOfGas.Error() {
				return nil, fmt.Errorf("gas required exceeds allowance (%d)", gasCap)
			}

			return nil, fmt.Errorf("estimate gas VMError: %s", result.VmError)
		}
	}

	return &evm.EstimateGasResponse{Gas: hi}, nil
}

// TraceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func (k Keeper) TraceTx(
	goCtx context.Context, req *evm.QueryTraceTxRequest,
) (*evm.QueryTraceTxResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// get the context of block beginning
	// 0 is a special value in `ContextWithHeight`
	contextHeight := max(req.BlockNumber, 1)

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx = ctx.WithBlockHeight(contextHeight)
	ctx = ctx.WithBlockTime(req.BlockTime)
	ctx = ctx.WithHeaderHash(gethcommon.Hex2Bytes(req.BlockHash))

	// to get the base fee we only need the block max gas in the consensus params
	ctx = ctx.WithConsensusParams(&cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{MaxGas: req.BlockMaxGas},
	})

	evmCfg := k.GetEVMConfig(ctx)
	// compute and use base fee of the height that is being traced
	baseFeeWeiPerGas := k.BaseFeeWeiPerGas(ctx)
	if baseFeeWeiPerGas != nil {
		evmCfg.BaseFeeWei = baseFeeWeiPerGas
	}

	signer := gethcore.MakeSigner(
		evmCfg.ChainConfig,
		big.NewInt(ctx.BlockHeight()),
		evm.ParseBlockTimeUnixU64(ctx),
	)
	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes()))

	// gas used at this point corresponds to GetProposerAddress &
	// CalculateBaseFee need to reset gas meter per transaction to be consistent
	// with tx execution and avoid stacking the gas used of every predecessor in
	// the same gas meter
	for i, tx := range req.Predecessors {
		ethTx := tx.AsTransaction()
		msg, err := core.TransactionToMessage(ethTx, signer, evmCfg.BaseFeeWei)
		if err != nil {
			continue
		}
		txConfig.TxHash = ethTx.Hash()
		txConfig.TxIndex = uint(i)
		// reset gas meter for each transaction
		ctx = ctx.WithGasMeter(eth.NewInfiniteGasMeterWithLimit(msg.GasLimit)).
			WithKVGasConfig(storetypes.GasConfig{}).
			WithTransientKVGasConfig(storetypes.GasConfig{})
		stateDB := statedb.New(ctx, &k, txConfig)
		evmObj := k.NewEVM(ctx, *msg, evmCfg, nil /*tracer*/, stateDB)
		rsp, err := k.ApplyEvmMsg(ctx, *msg, evmObj, false /*commit*/, txConfig.TxHash)
		if err != nil {
			continue
		}
		txConfig.LogIndex += uint(len(rsp.Logs))
	}

	tx := req.Msg.AsTransaction()
	txConfig.TxHash = tx.Hash()
	if len(req.Predecessors) > 0 {
		txConfig.TxIndex++
	}

	var tracerConfig json.RawMessage
	if req.TraceConfig != nil && req.TraceConfig.TracerConfig != nil {
		// ignore error. default to no traceConfig
		tracerConfig, _ = json.Marshal(req.TraceConfig.TracerConfig)
	}

	msg, err := core.TransactionToMessage(tx, signer, evmCfg.BaseFeeWei)
	if err != nil {
		return nil, err
	}

	result, _, err := k.TraceEthTxMsg(ctx, evmCfg, txConfig, *msg, req.TraceConfig, tracerConfig)
	if err != nil {
		// error will be returned with detail status from traceTx
		return nil, err
	}

	resultJson, err := json.Marshal(result)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	return &evm.QueryTraceTxResponse{
		Data: resultJson,
	}, nil
}

// TraceCall configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func (k Keeper) TraceCall(
	goCtx context.Context, req *evm.QueryTraceTxRequest,
) (*evm.QueryTraceTxResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// get the context of block beginning
	// 0 is a special value in `ContextWithHeight`
	contextHeight := max(req.BlockNumber, 1)

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx = ctx.WithBlockHeight(contextHeight)
	ctx = ctx.WithBlockTime(req.BlockTime)
	ctx = ctx.WithHeaderHash(gethcommon.Hex2Bytes(req.BlockHash))

	// to get the base fee we only need the block max gas in the consensus params
	ctx = ctx.WithConsensusParams(&cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{MaxGas: req.BlockMaxGas},
	})

	evmCfg := k.GetEVMConfig(ctx)

	// compute and use base fee of the height that is being traced
	baseFeeMicronibi := k.BaseFeeMicronibiPerGas(ctx)
	if baseFeeMicronibi != nil {
		evmCfg.BaseFeeWei = baseFeeMicronibi
	}

	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes()))

	var tracerConfig json.RawMessage
	if req.TraceConfig != nil && req.TraceConfig.TracerConfig != nil {
		// ignore error. default to no traceConfig
		tracerConfig, _ = json.Marshal(req.TraceConfig.TracerConfig)
	}

	// req.Msg is not signed, so to gethcore.Message because it's not signed and will fail on getting
	msgEthTx := req.Msg
	txData, err := evm.UnpackTxData(req.Msg.Data)
	if err != nil {
		return nil, grpcstatus.Errorf(grpccodes.Internal, "failed to unpack tx data: %s", err.Error())
	}
	evmMsg := core.Message{
		To:               txData.GetTo(),
		From:             gethcommon.HexToAddress(msgEthTx.From),
		Nonce:            txData.GetNonce(),
		Value:            txData.GetValueWei(), // amount
		GasLimit:         txData.GetGas(),
		GasPrice:         txData.GetGasPrice(),
		GasFeeCap:        txData.GetGasFeeCapWei(),
		GasTipCap:        txData.GetGasTipCapWei(),
		Data:             txData.GetData(),
		AccessList:       txData.GetAccessList(),
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	result, _, err := k.TraceEthTxMsg(ctx, evmCfg, txConfig, evmMsg, req.TraceConfig, tracerConfig)
	if err != nil {
		// error will be returned with detail status from traceTx
		return nil, err
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	return &evm.QueryTraceTxResponse{
		Data: resultData,
	}, nil
}

// Re-export of the default tracer timeout from go-ethereum.
// See "geth/eth/tracers/api.go".
const DefaultGethTraceTimeout = 5 * time.Second

// TraceBlock: Implements the gRPC query for "/eth.evm.v1.Query/TraceBlock".
// Configures a Nibiru EVM tracer that is used to "trace" and analyze
// the execution of transactions within a given block. Block information is read
// from the context (goCtx). [TraceBlock] is responsible iterates over each Eth
// transaction message and calls [TraceEthTxMsg] on it.
func (k Keeper) TraceBlock(
	goCtx context.Context, req *evm.QueryTraceBlockRequest,
) (*evm.QueryTraceBlockResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// get the context of block beginning
	// 0 is a special value in `ContextWithHeight`
	contextHeight := max(req.BlockNumber, 1)

	ctx := sdk.UnwrapSDKContext(goCtx).
		WithBlockHeight(contextHeight).
		WithBlockTime(req.BlockTime).
		WithHeaderHash(gethcommon.Hex2Bytes(req.BlockHash)).
		// to get the base fee we only need the block max gas in the consensus params
		WithConsensusParams(&cmtproto.ConsensusParams{
			Block: &cmtproto.BlockParams{MaxGas: req.BlockMaxGas},
		})

	evmCfg := k.GetEVMConfig(ctx)

	// compute and use base fee of height that is being traced
	if baseFeeMicronibiPerGas := k.BaseFeeMicronibiPerGas(ctx); baseFeeMicronibiPerGas != nil {
		baseFeeWeiPerGas := evm.NativeToWei(baseFeeMicronibiPerGas)
		evmCfg.BaseFeeWei = baseFeeWeiPerGas
	}
	var tracerConfig json.RawMessage
	if req.TraceConfig != nil && req.TraceConfig.TracerConfig != nil {
		// ignore error. default to no traceConfig
		tracerConfig, _ = json.Marshal(req.TraceConfig.TracerConfig)
	}

	signer := gethcore.MakeSigner(
		evmCfg.ChainConfig,
		big.NewInt(ctx.BlockHeight()),
		evm.ParseBlockTimeUnixU64(ctx),
	)
	txsLength := len(req.Txs)
	results := make([]*evm.TxTraceResult, 0, txsLength)

	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes()))

	for i, tx := range req.Txs {
		result := evm.TxTraceResult{}
		ethTx := tx.AsTransaction()
		txConfig.TxHash = ethTx.Hash()
		txConfig.TxIndex = uint(i)
		msg, err := core.TransactionToMessage(ethTx, signer, evmCfg.BaseFeeWei)
		if err != nil {
			result.Error = err.Error()
			continue
		}
		traceResult, logIndex, err := k.TraceEthTxMsg(ctx, evmCfg, txConfig, *msg, req.TraceConfig, tracerConfig)
		if err != nil {
			result.Error = err.Error()
		} else {
			txConfig.LogIndex = logIndex
			result.Result = traceResult
		}
		results = append(results, &result)
	}

	resultData, err := json.Marshal(results)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	return &evm.QueryTraceBlockResponse{
		Data: resultData,
	}, nil
}

// gasRemainingTxPartial returns a [gethcore.Transaction] that only has its "Gas"
// field set.
func gasRemainingTxPartial(gasLimit uint64) *gethcore.Transaction {
	txData := gethcore.LegacyTx{Gas: gasLimit}
	return gethcore.NewTx(&txData)
}

var gethTracerNames = set.New(
	"callTracer",     // Tracer with structured call tracer and hierarchical execution
	"flatCallTracer", // Similar to "callTracer" but with a flattened call trace
	"noopTracer",     // minimal tracer that doesn't actually collect data
	"4byteTracer",    // Collects statistics on 4-byte func signatures
	"muxTracer",      // A tracer that can combine multiple tracers in parallel
	"prestateTracer", // Captures the state of the tx before execution (pre-state)
	// Geth's StructLogger. It's not registered in the sense of
	// "go-ethereum/eth/tracers/native", meaning it cannot be accessed with
	// the [tracers.DefaultDirectory].New function.
	evm.TracerStruct,
)

// TraceEthTxMsg do trace on one transaction, it returns a tuple: (traceResult,
// nextLogIndex, error).
func (k *Keeper) TraceEthTxMsg(
	ctx sdk.Context,
	evmCfg statedb.EVMConfig,
	txConfig statedb.TxConfig,
	msg core.Message,
	traceConfig *evm.TraceConfig,
	tracerJSONConfig json.RawMessage,
) (traceResult *json.RawMessage, nextLogIndex uint, err error) {
	// Assemble the structured logger or the JavaScript tracerHooks
	var (
		tracer    *tracers.Tracer
		overrides *gethparams.ChainConfig
		timeout   = DefaultGethTraceTimeout
	)
	if traceConfig == nil {
		traceConfig = &evm.TraceConfig{}
	}

	logConfig := logger.Config{
		EnableMemory:     traceConfig.EnableMemory,
		DisableStorage:   traceConfig.DisableStorage,
		DisableStack:     traceConfig.DisableStack,
		EnableReturnData: traceConfig.EnableReturnData,
		Debug:            traceConfig.Debug,
		Limit:            int(traceConfig.Limit),
		Overrides:        overrides,
	}

	tCtx := &tracers.Context{
		BlockHash: txConfig.BlockHash,
		TxIndex:   int(txConfig.TxIndex),
		TxHash:    txConfig.TxHash,
	}

	var usingCallTracer bool
	if traceConfig.Tracer == evm.TracerStruct {
		logger := logger.NewStructLogger(&logConfig)
		tracer = &tracers.Tracer{
			Hooks:     logger.Hooks(),
			GetResult: logger.GetResult,
			Stop:      logger.Stop,
		}
	} else {
		if traceConfig.Tracer == "" || !gethTracerNames.Has(traceConfig.Tracer) {
			traceConfig.Tracer = "callTracer"
			usingCallTracer = true
		}
		tracer, err = tracers.DefaultDirectory.New(
			traceConfig.Tracer, tCtx, tracerJSONConfig, evmCfg.ChainConfig,
		)
		if err != nil {
			return nil, 0, grpcstatus.Error(grpccodes.Internal, err.Error())
		}
	}
	if tracer == nil && !usingCallTracer {
		traceConfig.Tracer = "callTracer"
		tracer, err = tracers.DefaultDirectory.New(
			traceConfig.Tracer, tCtx, tracerJSONConfig, evmCfg.ChainConfig,
		)
		if err != nil {
			return nil, 0, grpcstatus.Error(grpccodes.Internal, err.Error())
		}
	}

	// Define a meaningful timeout of a single transaction trace
	if traceConfig.Timeout != "" {
		if timeout, err = time.ParseDuration(traceConfig.Timeout); err != nil {
			return nil, 0, grpcstatus.Errorf(grpccodes.InvalidArgument, "timeout value: %s", err.Error())
		}
	}

	// Handle timeouts and RPC cancellations
	deadlineCtx, cancel := context.WithTimeout(ctx.Context(), timeout)
	defer cancel()

	go func() {
		<-deadlineCtx.Done()
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
		}
	}()

	// In order to be on in sync with the tx execution gas meter,
	// we need to:
	// 1. Reset GasMeter with InfiniteGasMeterWithLimit
	// 2. Setup an empty KV gas config for gas to be calculated by opcodes
	// and not kvstore actions
	// 3. Setup an empty transient KV gas config for transient gas to be
	// calculated by opcodes
	ctx = ctx.WithGasMeter(eth.NewInfiniteGasMeterWithLimit(msg.GasLimit)).
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})
	stateDB := statedb.New(ctx, k, txConfig)
	evmObj := k.NewEVM(ctx, msg, evmCfg, tracer.Hooks, stateDB)
	res, err := k.ApplyEvmMsg(ctx, msg, evmObj, false /*commit*/, txConfig.TxHash)
	if err != nil {
		return nil, 0, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	result, err := tracer.GetResult()
	if err != nil {
		return nil, 0, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	return &result, txConfig.LogIndex + uint(len(res.Logs)), nil
}

func (k Keeper) FunTokenMapping(
	goCtx context.Context, req *evm.QueryFunTokenMappingRequest,
) (*evm.QueryFunTokenMappingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// first try lookup by cosmos denom
	bankDenomIter := k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, req.Token)
	funTokenMappings := k.FunTokens.Collect(ctx, bankDenomIter)
	if len(funTokenMappings) > 0 {
		// assumes that there is only one mapping for a given denom
		return &evm.QueryFunTokenMappingResponse{
			FunToken: &funTokenMappings[0],
		}, nil
	}

	erc20AddrIter := k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, gethcommon.HexToAddress(req.Token))
	funTokenMappings = k.FunTokens.Collect(ctx, erc20AddrIter)
	if len(funTokenMappings) > 0 {
		// assumes that there is only one mapping for a given erc20 address
		return &evm.QueryFunTokenMappingResponse{
			FunToken: &funTokenMappings[0],
		}, nil
	}

	return nil, grpcstatus.Errorf(grpccodes.NotFound, "token mapping not found for %s", req.Token)
}
