// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm/statedb"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/evm"
)

// Compile-time interface assertion
var _ evm.QueryServer = Keeper{}

// EthAccount: Implements the gRPC query for "/eth.evm.v1.Query/EthAccount".
// EthAccount retrieves the account details for a given Ethereum hex address.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: Request containing the Ethereum hexadecimal address.
//
// Returns:
//   - A pointer to the QueryEthAccountResponse object containing the account details.
//   - An error if the account retrieval process encounters any issues.
func (k Keeper) EthAccount(
	goCtx context.Context, req *evm.QueryEthAccountRequest,
) (*evm.QueryEthAccountResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	addr := gethcommon.HexToAddress(req.Address)
	ctx := sdk.UnwrapSDKContext(goCtx)
	acct := k.GetAccountOrEmpty(ctx, addr)

	return &evm.QueryEthAccountResponse{
		Balance:  acct.Balance.String(),
		CodeHash: gethcommon.BytesToHash(acct.CodeHash).Hex(),
		Nonce:    acct.Nonce,
	}, nil
}

// NibiruAccount: Implements the gRPC query for "/eth.evm.v1.Query/NibiruAccount".
// NibiruAccount retrieves the Cosmos account details for a given Ethereum address.
//
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: The QueryNibiruAccountRequest object containing the Ethereum address.
//
// Returns:
//   - A pointer to the QueryNibiruAccountResponse object containing the Cosmos account details.
//   - An error if the account retrieval process encounters any issues.
func (k Keeper) NibiruAccount(
	goCtx context.Context, req *evm.QueryNibiruAccountRequest,
) (resp *evm.QueryNibiruAccountResponse, err error) {
	if err := req.Validate(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ethAddr := gethcommon.HexToAddress(req.Address)
	nibiruAddr := sdk.AccAddress(ethAddr.Bytes())

	accountOrNil := k.accountKeeper.GetAccount(ctx, nibiruAddr)
	resp = &evm.QueryNibiruAccountResponse{
		Address: nibiruAddr.String(),
	}

	if accountOrNil != nil {
		resp.Sequence = accountOrNil.GetSequence()
		resp.AccountNumber = accountOrNil.GetAccountNumber()
	}

	return resp, nil
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
// Balance retrieves the balance of an Ethereum address in "Ether", which
// actually refers to NIBI tokens on Nibiru EVM.
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
		Balance: balanceInt.String(),
	}, nil
}

// BaseFee implements the Query/BaseFee gRPC method
func (k Keeper) BaseFee(
	goCtx context.Context, _ *evm.QueryBaseFeeRequest,
) (*evm.QueryBaseFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := k.GetParams(ctx)
	ethCfg := params.ChainConfig.EthereumConfig(k.EthChainID(ctx))
	baseFee := k.GetBaseFee(ctx, ethCfg)

	res := &evm.QueryBaseFeeResponse{}
	if baseFee != nil {
		aux := sdkmath.NewIntFromBigInt(baseFee)
		res.BaseFee = &aux
	}
	return res, nil
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
	acct := k.GetAccountWithoutBalance(ctx, address)

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
// Parameters:
//   - goCtx: The context.Context object representing the request context.
//   - req: The EthCallRequest object containing the call parameters.
//
// Returns:
//   - A pointer to the MsgEthereumTxResponse object containing the result of the eth_call.
//   - An error if the eth_call process encounters any issues.
func (k Keeper) EthCall(
	goCtx context.Context, req *evm.EthCallRequest,
) (*evm.MsgEthereumTxResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	var args evm.JsonTxArgs
	err := json.Unmarshal(req.Args, &args)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	chainID, err := getChainID(ctx, req.ChainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	cfg, err := k.EVMConfig(ctx, GetProposerAddress(ctx, req.ProposerAddress), chainID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// ApplyMessageWithConfig expect correct nonce set in msg
	nonce := k.GetNonce(ctx, args.GetFrom())
	args.Nonce = (*hexutil.Uint64)(&nonce)

	msg, err := args.ToMessage(req.GasCap, cfg.BaseFee)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	txConfig := statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash()))

	// pass false to not commit StateDB
	res, err := k.ApplyEvmMsg(ctx, msg, nil, false, cfg, txConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return res, nil
}

// EstimateGas: Implements the gRPC query for "/eth.evm.v1.Query/EstimateGas".
// EstimateGas implements eth_estimateGas rpc api.
func (k Keeper) EstimateGas(
	goCtx context.Context, req *evm.EthCallRequest,
) (*evm.EstimateGasResponse, error) {
	// TODO: feat(evm): impl query EstimateGas
	return k.EstimateGasForEvmCallType(goCtx, req, evm.CallTypeRPC)
}

// EstimateGasForEvmCallType estimates the gas cost of a transaction. This can be
// called with the "eth_estimateGas" JSON-RPC method or an smart contract query.
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
	// TODO: feat(evm): impl query EstimateGasForEvmCallType
	return &evm.EstimateGasResponse{
		Gas: 220000, // TODO: replace with real gas calc
	}, nil
}

// TraceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func (k Keeper) TraceTx(
	goCtx context.Context, req *evm.QueryTraceTxRequest,
) (*evm.QueryTraceTxResponse, error) {
	// TODO: feat(evm): impl query TraceTx
	return &evm.QueryTraceTxResponse{
		Data: []byte{},
	}, common.ErrNotImplementedGprc()
}

// TraceBlock: Implements the gRPC query for "/eth.evm.v1.Query/TraceBlock".
// Configures a Nibiru EVM tracer that is used to "trace" and analyze
// the execution of transactions within a given block. Block information is read
// from the context (goCtx). [TraceBlock] is responsible iterates over each Eth
// transacion message and calls [TraceEthTxMsg] on it.
func (k Keeper) TraceBlock(
	goCtx context.Context, req *evm.QueryTraceBlockRequest,
) (*evm.QueryTraceBlockResponse, error) {
	// TODO: feat(evm): impl query TraceBlock
	return &evm.QueryTraceBlockResponse{
		Data: []byte{},
	}, common.ErrNotImplementedGprc()
}

// getChainID parse chainID from current context if not provided
func getChainID(ctx sdk.Context, chainID int64) (*big.Int, error) {
	if chainID == 0 {
		return eth.ParseEthChainID(ctx.ChainID())
	}
	return big.NewInt(chainID), nil
}

// GetProposerAddress returns current block proposer's address when provided proposer address is empty.
func GetProposerAddress(ctx sdk.Context, proposerAddress sdk.ConsAddress) sdk.ConsAddress {
	if len(proposerAddress) == 0 {
		proposerAddress = ctx.BlockHeader().ProposerAddress
	}
	return proposerAddress
}
