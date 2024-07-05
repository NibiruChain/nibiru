// Copyright (c) 2023-2024 Nibi, Inc.
package backend

import (
	"context"
	"math/big"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	params "github.com/ethereum/go-ethereum/params"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/NibiruChain/nibiru/app/server/config"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/rpc"
	"github.com/NibiruChain/nibiru/x/evm"
)

// BackendI implements the Cosmos and EVM backend.
type BackendI interface { //nolint: revive
	CosmosBackend
	EVMBackend
}

// CosmosBackend: Currently unused. Backend functionality for the shared
// "cosmos" RPC namespace. Implements [BackendI] in combination with [EVMBackend].
// TODO: feat(eth): Implement the cosmos JSON-RPC defined by Wallet Connect V2:
// https://docs.walletconnect.com/2.0/json-rpc/cosmos.
type CosmosBackend interface {
	// TODO: GetAccounts()
	// TODO: SignDirect()
	// TODO: SignAmino()
}

// EVMBackend implements the functionality shared within ethereum namespaces
// as defined by EIP-1474: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1474.md
// Implemented by Backend.
type EVMBackend interface {
	// Node specific queries
	Accounts() ([]common.Address, error)
	Syncing() (interface{}, error)
	RPCGasCap() uint64            // global gas cap for eth_call over rpc: DoS protection
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	RPCTxFeeCap() float64         // RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for send-transaction variants. The unit is ether.
	RPCMinGasPrice() int64

	// Sign Tx
	Sign(address common.Address, data hexutil.Bytes) (hexutil.Bytes, error)
	SendTransaction(args evm.JsonTxArgs) (common.Hash, error)
	SignTypedData(address common.Address, typedData apitypes.TypedData) (hexutil.Bytes, error)

	// Blocks Info
	BlockNumber() (hexutil.Uint64, error)
	GetBlockByNumber(blockNum rpc.BlockNumber, fullTx bool) (map[string]interface{}, error)
	GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error)
	GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint
	GetBlockTransactionCountByNumber(blockNum rpc.BlockNumber) *hexutil.Uint
	TendermintBlockByNumber(blockNum rpc.BlockNumber) (*tmrpctypes.ResultBlock, error)
	TendermintBlockResultByNumber(height *int64) (*tmrpctypes.ResultBlockResults, error)
	TendermintBlockByHash(blockHash common.Hash) (*tmrpctypes.ResultBlock, error)
	BlockNumberFromTendermint(blockNrOrHash rpc.BlockNumberOrHash) (rpc.BlockNumber, error)
	BlockNumberFromTendermintByHash(blockHash common.Hash) (*big.Int, error)
	EthMsgsFromTendermintBlock(block *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults) []*evm.MsgEthereumTx
	BlockBloom(blockRes *tmrpctypes.ResultBlockResults) (gethcore.Bloom, error)
	HeaderByNumber(blockNum rpc.BlockNumber) (*gethcore.Header, error)
	HeaderByHash(blockHash common.Hash) (*gethcore.Header, error)
	RPCBlockFromTendermintBlock(resBlock *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults, fullTx bool) (map[string]interface{}, error)
	EthBlockByNumber(blockNum rpc.BlockNumber) (*gethcore.Block, error)
	EthBlockFromTendermintBlock(resBlock *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults) (*gethcore.Block, error)

	// Account Info
	GetCode(address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error)
	GetBalance(address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error)
	GetStorageAt(address common.Address, key string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error)
	GetProof(address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*rpc.AccountResult, error)
	GetTransactionCount(address common.Address, blockNum rpc.BlockNumber) (*hexutil.Uint64, error)

	// Chain Info
	ChainID() (*hexutil.Big, error)
	ChainConfig() *params.ChainConfig
	// TODO: feat: Dynamic fees
	BaseFee(blockRes *tmrpctypes.ResultBlockResults) (*big.Int, error)
	CurrentHeader() (*gethcore.Header, error)
	PendingTransactions() ([]*sdk.Tx, error)
	FeeHistory(blockCount gethrpc.DecimalOrHex, lastBlock gethrpc.BlockNumber, rewardPercentiles []float64) (*rpc.FeeHistoryResult, error)
	SuggestGasTipCap(baseFee *big.Int) (*big.Int, error)

	// Tx Info
	GetTransactionByHash(txHash common.Hash) (*rpc.EthTxJsonRPC, error)
	GetTxByEthHash(txHash common.Hash) (*eth.TxResult, error)
	GetTxByTxIndex(height int64, txIndex uint) (*eth.TxResult, error)
	GetTransactionByBlockAndIndex(block *tmrpctypes.ResultBlock, idx hexutil.Uint) (*rpc.EthTxJsonRPC, error)
	GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error)
	GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpc.EthTxJsonRPC, error)
	GetTransactionByBlockNumberAndIndex(blockNum rpc.BlockNumber, idx hexutil.Uint) (*rpc.EthTxJsonRPC, error)

	// Send Transaction
	Resend(args evm.JsonTxArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error)
	SendRawTransaction(data hexutil.Bytes) (common.Hash, error)
	SetTxDefaults(args evm.JsonTxArgs) (evm.JsonTxArgs, error)
	EstimateGas(args evm.JsonTxArgs, blockNrOptional *rpc.BlockNumber) (hexutil.Uint64, error)
	DoCall(args evm.JsonTxArgs, blockNr rpc.BlockNumber) (*evm.MsgEthereumTxResponse, error)
	GasPrice() (*hexutil.Big, error)

	// Filter API
	GetLogs(hash common.Hash) ([][]*gethcore.Log, error)
	GetLogsByHeight(height *int64) ([][]*gethcore.Log, error)
	BloomStatus() (uint64, uint64)

	// Tracing
	TraceTransaction(hash common.Hash, config *evm.TraceConfig) (interface{}, error)
	TraceBlock(
		height rpc.BlockNumber,
		config *evm.TraceConfig,
		block *tmrpctypes.ResultBlock,
	) ([]*evm.TxTraceResult, error)
}

var _ BackendI = (*Backend)(nil)

// Backend implements the BackendI interface
type Backend struct {
	ctx                 context.Context
	clientCtx           client.Context
	queryClient         *rpc.QueryClient // gRPC query client
	logger              log.Logger
	chainID             *big.Int
	cfg                 config.Config
	allowUnprotectedTxs bool
	indexer             eth.EVMTxIndexer
}

// NewBackend creates a new Backend instance for cosmos and ethereum namespaces
func NewBackend(
	ctx *server.Context,
	logger log.Logger,
	clientCtx client.Context,
	allowUnprotectedTxs bool,
	indexer eth.EVMTxIndexer,
) *Backend {
	chainID, err := eth.ParseEthChainID(clientCtx.ChainID)
	if err != nil {
		panic(err)
	}

	appConf, err := config.GetConfig(ctx.Viper)
	if err != nil {
		panic(err)
	}

	return &Backend{
		ctx:                 context.Background(),
		clientCtx:           clientCtx,
		queryClient:         rpc.NewQueryClient(clientCtx),
		logger:              logger.With("module", "backend"),
		chainID:             chainID,
		cfg:                 appConf,
		allowUnprotectedTxs: allowUnprotectedTxs,
		indexer:             indexer,
	}
}
