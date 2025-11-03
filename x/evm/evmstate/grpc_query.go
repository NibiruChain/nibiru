// Copyright (c) 2023-2024 Nibi, Inc.
package evmstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	sdkmath "cosmossdk.io/math"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/nutil/set"

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
	addrBech32, err := req.Validate()
	if err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	balWei := k.Bank.GetWeiBalance(ctx, addrBech32)

	var addrEthHex string
	acct := NewEmptyAccount()
	if len(addrBech32.Bytes()) == appconst.ADDR_LEN_EOA {
		addrEth := eth.NibiruAddrToEthAddr(addrBech32)
		acctMaybe := k.GetAccount(ctx, addrEth)
		if acctMaybe != nil {
			acct = acctMaybe
		}
		addrEthHex = addrEth.Hex()
	}

	return &evm.QueryEthAccountResponse{
		EthAddress:    addrEthHex,
		Bech32Address: addrBech32.String(),
		BalanceWei:    balWei.String(),
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
//   - req: The QueryBalanceRequest object containing the Ethereum hex address or
//     nibi-prefixed Bech32 address.
//
// Returns:
//   - A pointer to the QueryBalanceResponse object containing the balance.
//   - An error if the balance retrieval process encounters any issues.
func (k Keeper) Balance(
	goCtx context.Context,
	req *evm.QueryBalanceRequest,
) (*evm.QueryBalanceResponse, error) {
	addrBech32, err := req.Validate()
	if err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &evm.QueryBalanceResponse{
		BalanceWei: k.Bank.GetWeiBalance(ctx, addrBech32).String(),
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
	ctx = ctx.WithValue(evm.CtxKeyEvmSimulation, true)

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

	// pass false to not commit StateDB
	txConfig := NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash()))
	sdb := NewSDB(ctx, k, txConfig)
	evm := k.NewEVM(ctx, msg, evmCfg, nil /*tracer*/, sdb)
	res, err := k.ApplyEvmMsg(msg, evm, false /*commit*/)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	return res, nil
}

// EstimateGas: Implements the gRPC query for "/eth.evm.v1.Query/EstimateGas".
// This estimates the lowest possible gas limit that allows a transaction to run
// successfully with the provided context options. This can be called with the
// "eth_estimateGas" JSON-RPC method.
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
func (k Keeper) EstimateGas(
	goCtx context.Context, req *evm.EthCallRequest,
) (*evm.EstimateGasResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	rootCtx := sdk.UnwrapSDKContext(goCtx).
		WithValue(evm.CtxKeyEvmSimulation, true)
	evmCfg := k.GetEVMConfig(rootCtx)

	if req.GasCap < gethparams.TxGas {
		return nil, grpcstatus.Errorf(grpccodes.InvalidArgument, "gas cap cannot be lower than %d", gethparams.TxGas)
	}

	var args evm.JsonTxArgs
	err := json.Unmarshal(req.Args, &args)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, err.Error())
	}

	// ApplyMessageWithConfig expect correct nonce set in msg
	nonce := k.GetAccNonce(rootCtx, args.GetFrom())
	args.Nonce = (*hexutil.Uint64)(&nonce)

	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		// Set smart lower bound based on the gas used in the first execution
		// (base case).
		lo uint64
		hi uint64

		// executable runs one probe at a specific gas limit.
		//   - Rewrites evmMsg.GasLimit to the probed value.
		//   - Constructs a fresh SDB on a context with an infinite gas meter and zero
		//     KV/transient KV gas costs, isolating the probe from store-gas panics.
		//   - Defers a panic classifier, where SDK/go-ethereum "out of gas"
		//     panics result in  { vmError=true, err=nil }. Any other panic gets
		//     bubbled up through the call stack.
		//   - Returns (vmError, resp, err) where vmError signals VM-level failure
		//     (incl. OOG/revert), and err signals consensus/unexpected failure.
		executable func(gas uint64) (vmError bool, rsp *evm.MsgEthereumTxResponse, err error)
	)

	// Determine the highest gas limit can be used during the estimation.
	// Start with block gas limit
	params := rootCtx.ConsensusParams()
	if params != nil && params.Block != nil && params.Block.MaxGas > 0 {
		hi = uint64(params.Block.MaxGas)
	} else {
		// Fallback to gasCap if block params not available
		hi = req.GasCap
	}

	// Override with user-provided gas limit if it's valid
	if args.Gas != nil && uint64(*args.Gas) >= gethparams.TxGas {
		hi = uint64(*args.Gas)
	}

	// Recap the highest gas allowance with specified gascap.
	if req.GasCap != 0 && hi > req.GasCap {
		hi = req.GasCap
	}

	// convert the tx args to an ethereum message
	evmMsg, err := args.ToMessage(req.GasCap, evmCfg.BaseFeeWei)
	if err != nil {
		return nil, grpcstatus.Error(grpccodes.Internal, err.Error())
	}

	executable = func(gas uint64) (vmError bool, rsp *evm.MsgEthereumTxResponse, err error) {
		defer func() {
			// Recover OOG panics as a normal VM failure so the binary search can
			// increase gas. Any non-OOG panic aborts the search with a
			// contextual error for diagnostics.
			var (
				oog  bool
				perr error
			)

			if panicInfo := recover(); panicInfo != nil {
				if _, isOutOfGasPanic := panicInfo.(sdk.ErrorOutOfGas); isOutOfGasPanic {
					oog, perr = true, vm.ErrOutOfGas
				} else if strings.Contains(fmt.Sprint(panicInfo), "out of gas") {
					oog, perr = true, vm.ErrOutOfGas
				} else {
					// Non-OOG panics are not handled here
					oog, perr = false, fmt.Errorf(
						`unexpected panic in eth_estimateGas { gas: %d }: %v`, gas, panicInfo)
				}
			}

			if oog {
				vmError, rsp, err = true, nil, nil
				return
			} else if perr != nil {
				err = perr // Unexpected panic -> Abort the search
				return
			}
		}()
		evmMsg = core.Message{ // update the message with the new gas value
			To:               evmMsg.To,
			From:             evmMsg.From,
			Nonce:            evmMsg.Nonce,
			Value:            evmMsg.Value,
			GasLimit:         gas, // <---- This one changes
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

		// Initialize SDB
		sdb := k.NewSDB(
			rootCtx,
			k.TxConfig(rootCtx, rootCtx.EvmTxHash()),
		)
		sdb.SetCtx(
			sdb.Ctx().
				WithGasMeter(eth.NewInfiniteGasMeterWithLimit(evmMsg.GasLimit)).
				WithKVGasConfig(storetypes.GasConfig{}).
				WithTransientKVGasConfig(storetypes.GasConfig{}),
		)

		acct := k.GetAccount(sdb.Ctx(), evmMsg.From)

		from := evmMsg.From
		if acct == nil {
			acc := k.accountKeeper.NewAccountWithAddress(sdb.Ctx(), from[:])
			k.accountKeeper.SetAccount(sdb.Ctx(), acc)
			acct = NewEmptyAccount()
		}
		// When submitting a transaction, the `EthIncrementSenderSequence` ante handler increases the account nonce
		acct.Nonce = nonce + 1
		err = k.SetAccount(sdb.Ctx(), from, *acct)
		if err != nil {
			return true, nil, err
		}

		// pass false to not commit StateDB
		evmObj := k.NewEVM(sdb.Ctx(), evmMsg, evmCfg, nil /*tracer*/, sdb)
		rsp, err = k.ApplyEvmMsg(evmMsg, evmObj, false /*commit*/)
		if err != nil {
			if strings.Contains(err.Error(), core.ErrIntrinsicGas.Error()) {
				return true, nil, nil // Special case, raise gas limit
			}
			return true, nil, fmt.Errorf("error applying EVM message to StateDB: %w", err) // Bail out
		}
		return len(rsp.VmError) > 0, rsp, nil
	}

	// BASE CASE:  Jumping straight into binary search is extermely inefficient.
	// Instead, execute at the highest allowable gas limit first to validate and
	// set a smarter lower bound.
	failed, result, err := executable(hi)
	if err != nil {
		return nil, fmt.Errorf("eth call exec error: %w", err)
	}
	// If the base case fails for non-gas reasons, return the error immediately
	if failed {
		if result != nil && result.VmError != "" && result.VmError != vm.ErrOutOfGas.Error() {
			if result.VmError == vm.ErrExecutionReverted.Error() {
				return nil, fmt.Errorf("estimate gas VMError: %w", evm.NewRevertError(result.Ret))
			}
			return nil, fmt.Errorf("estimate gas VMError: %s", result.VmError)
		}
		return nil, fmt.Errorf("gas required exceeds allowance (%d)", hi)
	}

	// Set smart lower bound based on actual gas used
	if result.GasUsed > 0 {
		lo = result.GasUsed - 1
	} else {
		lo = 0
	}

	// Execute the binary search and hone in on an executable gas limit
	estimateTolerance := evm.GasEstimateErrorRatioTolerance
	if rootCtx.Value(evm.CtxKeyGasEstimateZeroTolerance) == true {
		estimateTolerance = 0.00
	}
	hi, err = evm.BinSearch(lo, hi, executable, estimateTolerance)
	if err != nil {
		return nil, err
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
	ctx = ctx.WithValue(evm.CtxKeyEvmSimulation, true)
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
	txConfig := NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes()))

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
		sdb := NewSDB(ctx, &k, txConfig)
		evmObj := k.NewEVM(ctx, *msg, evmCfg, nil /*tracer*/, sdb)
		rsp, err := k.ApplyEvmMsg(*msg, evmObj, false /*commit*/)
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
	ctx = ctx.WithValue(evm.CtxKeyEvmSimulation, true)
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

	txConfig := NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes()))

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
	ctx = ctx.WithValue(evm.CtxKeyEvmSimulation, true)

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

	// NOTE: Nibiru EVM uses exclusively native tracers and considers JS tracers
	// out of scope.
	//
	// Geth differentiates between native tracers and JS tracers.
	// Native tracers are the defaults like the "callTracer" and others have low
	// overhead and return errors if any of the txs fail tracing.
	//
	// JS tracers (geth only) have high overhead. Tracing for them runs a
	// parallel process that generates statesin one thread and traces txs in
	// separate worker threads. JS tracers store tracing errors for each tx as
	// fields of the returned trace result instead of failing the query.
	var (
		results  = make([]evm.TxTraceResult, len(req.Txs))
		txConfig = NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash().Bytes()))
		// Transaction data as an EVM message to be traced.
		msg *core.Message
	)
	for i, tx := range req.Txs {
		result := evm.TxTraceResult{}
		ethTx := tx.AsTransaction()
		result.TxHash = ethTx.Hash()
		txConfig.TxHash = result.TxHash
		txConfig.TxIndex = uint(i)
		// Here in "core.TransactionToMessage", the resulting msg is guaranteed
		// not to be nil, and potential signer errors are not relevant for
		// tracing, as this is only a query.
		msg, _ = core.TransactionToMessage(ethTx, signer, evmCfg.BaseFeeWei)
		traceResult, logIndex, err := k.TraceEthTxMsg(ctx, evmCfg, txConfig, *msg, req.TraceConfig, tracerConfig)
		if err != nil {
			// Since Nibiru uses native tracers from geth, failure to trace any
			// tx means block tracing fails too.
			return nil, fmt.Errorf("trace tx error { txhash: %s, blockHeight: %d }: %w", ethTx.Hash().Hex(), ctx.BlockHeight(), err)
		}
		txConfig.LogIndex = logIndex
		result.Result = traceResult
		results[i] = result
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
	evmCfg EVMConfig,
	txConfig TxConfig,
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
	sdb := NewSDB(ctx, k, txConfig)
	evmObj := k.NewEVM(ctx, msg, evmCfg, tracer.Hooks, sdb)
	res, err := k.ApplyEvmMsg(msg, evmObj, false /*commit*/)
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
