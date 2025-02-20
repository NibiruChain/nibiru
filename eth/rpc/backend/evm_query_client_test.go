package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/ethereum/go-ethereum/common"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/rpc"
	"github.com/NibiruChain/nibiru/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/x/evm"
	evmtest "github.com/NibiruChain/nibiru/x/evm/evmtest"
)

// QueryClient defines a mocked object that implements the ethermint GRPC
// QueryClient interface. It allows for performing QueryClient queries without having
// to run a ethermint GRPC server.
//
// To use a mock method it has to be registered in a given test.
var _ evm.QueryClient = &mocks.EVMQueryClient{}

func TEST_CHAIN_ID_NUMBER() int64 {
	n, _ := eth.ParseEthChainID(eth.EIP155ChainID_Testnet)
	return n.Int64()
}

// TraceTransaction
func RegisterTraceTransactionWithPredecessors(
	queryClient *mocks.EVMQueryClient, msgEthTx *evm.MsgEthereumTx, predecessors []*evm.MsgEthereumTx,
) {
	data := []byte{0x7b, 0x22, 0x74, 0x65, 0x73, 0x74, 0x22, 0x3a, 0x20, 0x22, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x22, 0x7d}
	queryClient.On("TraceTx", rpc.NewContextWithHeight(1),
		&evm.QueryTraceTxRequest{Msg: msgEthTx, BlockNumber: 1, Predecessors: predecessors, ChainId: TEST_CHAIN_ID_NUMBER(), BlockMaxGas: -1}).
		Return(&evm.QueryTraceTxResponse{Data: data}, nil)
}

func RegisterTraceTransaction(
	queryClient *mocks.EVMQueryClient, msgEthTx *evm.MsgEthereumTx,
) {
	data := []byte{0x7b, 0x22, 0x74, 0x65, 0x73, 0x74, 0x22, 0x3a, 0x20, 0x22, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x22, 0x7d}
	queryClient.On("TraceTx", rpc.NewContextWithHeight(1), &evm.QueryTraceTxRequest{Msg: msgEthTx, BlockNumber: 1, ChainId: TEST_CHAIN_ID_NUMBER(), BlockMaxGas: -1}).
		Return(&evm.QueryTraceTxResponse{Data: data}, nil)
}

func RegisterTraceTransactionError(
	queryClient *mocks.EVMQueryClient, msgEthTx *evm.MsgEthereumTx,
) {
	queryClient.On("TraceTx", rpc.NewContextWithHeight(1), &evm.QueryTraceTxRequest{Msg: msgEthTx, BlockNumber: 1, ChainId: TEST_CHAIN_ID_NUMBER()}).
		Return(nil, errortypes.ErrInvalidRequest)
}

// TraceBlock
func RegisterTraceBlock(
	queryClient *mocks.EVMQueryClient, txs []*evm.MsgEthereumTx,
) {
	data := []byte{0x7b, 0x22, 0x74, 0x65, 0x73, 0x74, 0x22, 0x3a, 0x20, 0x22, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x22, 0x7d}
	queryClient.On("TraceBlock", rpc.NewContextWithHeight(1),
		&evm.QueryTraceBlockRequest{Txs: txs, BlockNumber: 1, TraceConfig: &evm.TraceConfig{}, ChainId: TEST_CHAIN_ID_NUMBER(), BlockMaxGas: -1}).
		Return(&evm.QueryTraceBlockResponse{Data: data}, nil)
}

func RegisterTraceBlockError(queryClient *mocks.EVMQueryClient) {
	queryClient.On("TraceBlock", rpc.NewContextWithHeight(1), &evm.QueryTraceBlockRequest{}).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Params
func RegisterParams(
	queryClient *mocks.EVMQueryClient, header *metadata.MD, height int64,
) {
	queryClient.On("Params", rpc.NewContextWithHeight(height), &evm.QueryParamsRequest{}, grpc.Header(header)).
		Return(&evm.QueryParamsResponse{}, nil).
		Run(func(args mock.Arguments) {
			// If Params call is successful, also update the header height
			arg := args.Get(2).(grpc.HeaderCallOption)
			h := metadata.MD{}
			h.Set(grpctypes.GRPCBlockHeightHeader, fmt.Sprint(height))
			*arg.HeaderAddr = h
		})
}

func RegisterParamsWithoutHeader(
	queryClient *mocks.EVMQueryClient, height int64,
) {
	queryClient.On("Params", rpc.NewContextWithHeight(height), &evm.QueryParamsRequest{}).
		Return(&evm.QueryParamsResponse{Params: evm.DefaultParams()}, nil)
}

func RegisterParamsInvalidHeader(
	queryClient *mocks.EVMQueryClient, header *metadata.MD, height int64,
) {
	queryClient.On("Params", rpc.NewContextWithHeight(height), &evm.QueryParamsRequest{}, grpc.Header(header)).
		Return(&evm.QueryParamsResponse{}, nil).
		Run(func(args mock.Arguments) {
			// If Params call is successful, also update the header height
			arg := args.Get(2).(grpc.HeaderCallOption)
			h := metadata.MD{}
			*arg.HeaderAddr = h
		})
}

func RegisterParamsInvalidHeight(queryClient *mocks.EVMQueryClient, header *metadata.MD, height int64) {
	queryClient.On("Params", rpc.NewContextWithHeight(height), &evm.QueryParamsRequest{}, grpc.Header(header)).
		Return(&evm.QueryParamsResponse{}, nil).
		Run(func(args mock.Arguments) {
			// If Params call is successful, also update the header height
			arg := args.Get(2).(grpc.HeaderCallOption)
			h := metadata.MD{}
			h.Set(grpctypes.GRPCBlockHeightHeader, "invalid")
			*arg.HeaderAddr = h
		})
}

func RegisterParamsWithoutHeaderError(queryClient *mocks.EVMQueryClient, height int64) {
	queryClient.On("Params", rpc.NewContextWithHeight(height), &evm.QueryParamsRequest{}).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Params returns error
func RegisterParamsError(
	queryClient *mocks.EVMQueryClient, header *metadata.MD, height int64,
) {
	queryClient.On("Params", rpc.NewContextWithHeight(height), &evm.QueryParamsRequest{}, grpc.Header(header)).
		Return(nil, errortypes.ErrInvalidRequest)
}

func TestRegisterParams(t *testing.T) {
	var header metadata.MD
	queryClient := mocks.NewEVMQueryClient(t)

	height := int64(1)
	RegisterParams(queryClient, &header, height)

	_, err := queryClient.Params(rpc.NewContextWithHeight(height), &evm.QueryParamsRequest{}, grpc.Header(&header))
	require.NoError(t, err)
	blockHeightHeader := header.Get(grpctypes.GRPCBlockHeightHeader)
	headerHeight, err := strconv.ParseInt(blockHeightHeader[0], 10, 64)
	require.NoError(t, err)
	require.Equal(t, height, headerHeight)
}

func TestRegisterParamsError(t *testing.T) {
	queryClient := mocks.NewEVMQueryClient(t)
	RegisterBaseFeeError(queryClient)
	_, err := queryClient.BaseFee(rpc.NewContextWithHeight(1), &evm.QueryBaseFeeRequest{})
	require.Error(t, err)
}

// ETH Call
func RegisterEthCall(
	queryClient *mocks.EVMQueryClient, request *evm.EthCallRequest,
) {
	ctx, _ := context.WithCancel(rpc.NewContextWithHeight(1)) //nolint
	queryClient.On("EthCall", ctx, request).
		Return(&evm.MsgEthereumTxResponse{}, nil)
}

func RegisterEthCallError(
	queryClient *mocks.EVMQueryClient, request *evm.EthCallRequest,
) {
	ctx, _ := context.WithCancel(rpc.NewContextWithHeight(1)) //nolint
	queryClient.On("EthCall", ctx, request).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Estimate Gas
func RegisterEstimateGas(
	queryClient *mocks.EVMQueryClient, args evm.JsonTxArgs,
) {
	bz, _ := json.Marshal(args)
	queryClient.On("EstimateGas", rpc.NewContextWithHeight(1), &evm.EthCallRequest{Args: bz, ChainId: args.ChainID.ToInt().Int64()}).
		Return(&evm.EstimateGasResponse{}, nil)
}

// BaseFee
func RegisterBaseFee(
	queryClient *mocks.EVMQueryClient, baseFee math.Int,
) {
	queryClient.On("BaseFee", rpc.NewContextWithHeight(1), &evm.QueryBaseFeeRequest{}).
		Return(&evm.QueryBaseFeeResponse{BaseFee: &baseFee}, nil)
}

// Base fee returns error
func RegisterBaseFeeError(queryClient *mocks.EVMQueryClient) {
	queryClient.On("BaseFee", rpc.NewContextWithHeight(1), &evm.QueryBaseFeeRequest{}).
		Return(&evm.QueryBaseFeeResponse{}, evm.ErrInvalidBaseFee)
}

// Base fee not enabled
func RegisterBaseFeeDisabled(queryClient *mocks.EVMQueryClient) {
	queryClient.On("BaseFee", rpc.NewContextWithHeight(1), &evm.QueryBaseFeeRequest{}).
		Return(&evm.QueryBaseFeeResponse{}, nil)
}

func TestRegisterBaseFee(t *testing.T) {
	baseFee := math.NewInt(1)
	queryClient := mocks.NewEVMQueryClient(t)
	RegisterBaseFee(queryClient, baseFee)
	res, err := queryClient.BaseFee(rpc.NewContextWithHeight(1), &evm.QueryBaseFeeRequest{})
	require.Equal(t, &evm.QueryBaseFeeResponse{BaseFee: &baseFee}, res)
	require.NoError(t, err)
}

func TestRegisterBaseFeeError(t *testing.T) {
	queryClient := mocks.NewEVMQueryClient(t)
	RegisterBaseFeeError(queryClient)
	res, err := queryClient.BaseFee(rpc.NewContextWithHeight(1), &evm.QueryBaseFeeRequest{})
	require.Equal(t, &evm.QueryBaseFeeResponse{}, res)
	require.Error(t, err)
}

func TestRegisterBaseFeeDisabled(t *testing.T) {
	queryClient := mocks.NewEVMQueryClient(t)
	RegisterBaseFeeDisabled(queryClient)
	res, err := queryClient.BaseFee(rpc.NewContextWithHeight(1), &evm.QueryBaseFeeRequest{})
	require.Equal(t, &evm.QueryBaseFeeResponse{}, res)
	require.NoError(t, err)
}

// ValidatorAccount
func RegisterValidatorAccount(
	queryClient *mocks.EVMQueryClient, validator sdk.AccAddress,
) {
	queryClient.On("ValidatorAccount", rpc.NewContextWithHeight(1), &evm.QueryValidatorAccountRequest{}).
		Return(&evm.QueryValidatorAccountResponse{AccountAddress: validator.String()}, nil)
}

func RegisterValidatorAccountError(queryClient *mocks.EVMQueryClient) {
	queryClient.On("ValidatorAccount", rpc.NewContextWithHeight(1), &evm.QueryValidatorAccountRequest{}).
		Return(nil, status.Error(codes.InvalidArgument, "empty request"))
}

func TestRegisterValidatorAccount(t *testing.T) {
	queryClient := mocks.NewEVMQueryClient(t)

	validator := sdk.AccAddress(evmtest.NewEthAccInfo().EthAddr.Bytes())
	RegisterValidatorAccount(queryClient, validator)
	res, err := queryClient.ValidatorAccount(rpc.NewContextWithHeight(1), &evm.QueryValidatorAccountRequest{})
	require.Equal(t, &evm.QueryValidatorAccountResponse{AccountAddress: validator.String()}, res)
	require.NoError(t, err)
}

// Code
func RegisterCode(
	queryClient *mocks.EVMQueryClient, addr common.Address, code []byte,
) {
	queryClient.On("Code", rpc.NewContextWithHeight(1), &evm.QueryCodeRequest{Address: addr.String()}).
		Return(&evm.QueryCodeResponse{Code: code}, nil)
}

func RegisterCodeError(queryClient *mocks.EVMQueryClient, addr common.Address) {
	queryClient.On("Code", rpc.NewContextWithHeight(1), &evm.QueryCodeRequest{Address: addr.String()}).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Storage
func RegisterStorageAt(
	queryClient *mocks.EVMQueryClient, addr common.Address,
	key string, storage string,
) {
	queryClient.On("Storage", rpc.NewContextWithHeight(1), &evm.QueryStorageRequest{Address: addr.String(), Key: key}).
		Return(&evm.QueryStorageResponse{Value: storage}, nil)
}

func RegisterStorageAtError(
	queryClient *mocks.EVMQueryClient, addr common.Address, key string,
) {
	queryClient.On("Storage", rpc.NewContextWithHeight(1), &evm.QueryStorageRequest{Address: addr.String(), Key: key}).
		Return(nil, errortypes.ErrInvalidRequest)
}

func RegisterAccount(
	queryClient *mocks.EVMQueryClient, addr common.Address, height int64,
) {
	queryClient.On("EthAccount", rpc.NewContextWithHeight(height), &evm.QueryEthAccountRequest{Address: addr.String()}).
		Return(&evm.QueryEthAccountResponse{
			Balance:  "0",
			CodeHash: "",
			Nonce:    0,
		},
			nil,
		)
}

// Balance
func RegisterBalance(
	queryClient *mocks.EVMQueryClient, addr common.Address, height int64,
) {
	queryClient.On("Balance", rpc.NewContextWithHeight(height), &evm.QueryBalanceRequest{Address: addr.String()}).
		Return(&evm.QueryBalanceResponse{Balance: "1"}, nil)
}

func RegisterBalanceInvalid(
	queryClient *mocks.EVMQueryClient, addr common.Address, height int64,
) {
	queryClient.On("Balance", rpc.NewContextWithHeight(height), &evm.QueryBalanceRequest{Address: addr.String()}).
		Return(&evm.QueryBalanceResponse{Balance: "invalid"}, nil)
}

func RegisterBalanceNegative(
	queryClient *mocks.EVMQueryClient, addr common.Address, height int64,
) {
	queryClient.On("Balance", rpc.NewContextWithHeight(height), &evm.QueryBalanceRequest{Address: addr.String()}).
		Return(&evm.QueryBalanceResponse{Balance: "-1"}, nil)
}

func RegisterBalanceError(
	queryClient *mocks.EVMQueryClient, addr common.Address, height int64,
) {
	queryClient.On("Balance", rpc.NewContextWithHeight(height), &evm.QueryBalanceRequest{Address: addr.String()}).
		Return(nil, errortypes.ErrInvalidRequest)
}
