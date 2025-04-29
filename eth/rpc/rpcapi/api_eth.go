// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"context"

	gethmath "github.com/ethereum/go-ethereum/common/math"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	cmtlog "github.com/cometbft/cometbft/libs/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// TODO: Remove this interface over since it's largely unused and the expected
// API methods are tested in "TestExpectedMethods" in eth_api_test.go.
type IEthAPI interface {
	// Account Information
	//
	// Returns information regarding an address's stored on-chain data.
	GetBalance(
		address common.Address, blockNrOrHash rpc.BlockNumberOrHash,
	) (*hexutil.Big, error)
	GetStorageAt(
		address common.Address, key string, blockNrOrHash rpc.BlockNumberOrHash,
	) (hexutil.Bytes, error)
	GetCode(
		address common.Address, blockNrOrHash rpc.BlockNumberOrHash,
	) (hexutil.Bytes, error)
	GetProof(
		address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash,
	) (*rpc.AccountResult, error)

	// Chain Information
	//
	// Returns information on the Ethereum network and internal settings.
	ProtocolVersion() hexutil.Uint
	GasPrice() (*hexutil.Big, error)
	EstimateGas(
		args evm.JsonTxArgs, blockNrOptional *rpc.BlockNumber,
	) (hexutil.Uint64, error)
	FeeHistory(
		blockCount gethmath.HexOrDecimal64,
		lastBlock gethrpc.BlockNumber,
		rewardPercentiles []float64,
	) (*rpc.FeeHistoryResult, error)
	MaxPriorityFeePerGas() (*hexutil.Big, error)
	ChainId() (*hexutil.Big, error)

	// Getting Uncles
	//
	// Returns information on uncle blocks are which are network rejected blocks
	// and replaced by a canonical block instead.
	GetUncleByBlockHashAndIndex(
		hash common.Hash, idx hexutil.Uint,
	) map[string]any
	GetUncleByBlockNumberAndIndex(
		number, idx hexutil.Uint,
	) map[string]any
	GetUncleCountByBlockHash(hash common.Hash) hexutil.Uint
	GetUncleCountByBlockNumber(blockNum rpc.BlockNumber) hexutil.Uint

	// Other
	Syncing() (any, error)
	GetTransactionLogs(txHash common.Hash) ([]*gethcore.Log, error)
	FillTransaction(
		args evm.JsonTxArgs,
	) (*rpc.SignTransactionResult, error)
	GetPendingTransactions() ([]*rpc.EthTxJsonRPC, error)
}

var _ IEthAPI = (*EthAPI)(nil)

// EthAPI: Allows connection to a full node of the Nibiru blockchain
// network via Nibiru EVM. Developers can interact with on-chain EVM data and
// send different types of transactions to the network by utilizing the endpoints
// provided by the API.
//
// [EthAPI] contains much of the "eth_" prefixed methods in the Web3 JSON-RPC spec.
//
// The API follows a JSON-RPC standard. If not otherwise
// specified, the interface is derived from the Alchemy Ethereum API:
// https://docs.alchemy.com/alchemy/apis/ethereum
type EthAPI struct {
	ctx     context.Context
	logger  cmtlog.Logger
	backend *Backend
}

// NewImplEthAPI creates an instance of the public ETH Web3 API.
func NewImplEthAPI(logger cmtlog.Logger, backend *Backend) *EthAPI {
	api := &EthAPI{
		ctx:     context.Background(),
		logger:  logger.With("client", "json-rpc"),
		backend: backend,
	}

	return api
}

// --------------------------------------------------------------------------
//                           Blocks
// --------------------------------------------------------------------------

// BlockNumber returns the current block number.
func (e *EthAPI) BlockNumber() (hexutil.Uint64, error) {
	e.logger.Debug("eth_blockNumber")
	return e.backend.BlockNumber()
}

// GetBlockByNumber returns the block identified by number.
//   - When "fullTx" is true, all transactions in the block are returned, otherwise
//     the block will only show transaction hashes.
func (e *EthAPI) GetBlockByNumber(ethBlockNum rpc.BlockNumber, fullTx bool) (map[string]any, error) {
	methodName := "eth_getBlockByNumber"
	e.logger.Debug(methodName, "blockNumber", ethBlockNum, "fullTx", fullTx)
	block, err := e.backend.GetBlockByNumber(ethBlockNum, fullTx)
	logError(e.logger, err, methodName)
	return block, err
}

// GetBlockByHash returns the block identified by hash.
//   - When "fullTx" is true, all transactions in the block are returned, otherwise
//     the block will only show transaction hashes.
func (e *EthAPI) GetBlockByHash(
	blockHash common.Hash,
	fullTx bool,
) (map[string]any, error) {
	methodName := "eth_getBlockByHash"
	e.logger.Debug(methodName, "hash", blockHash.Hex(), "fullTx", fullTx)
	block, err := e.backend.GetBlockByHash(blockHash, fullTx)
	logError(e.logger, err, methodName)
	return block, err
}

// logError logs a backend error if one is present
func logError(logger cmtlog.Logger, err error, methodName string) {
	if err != nil {
		logger.Debug(methodName+" failed", "error", err.Error())
	}
}

// --------------------------------------------------------------------------
//                           Read Txs
// --------------------------------------------------------------------------

// GetTransactionByHash returns the Ethereum format transaction identified by
// Ethereum transaction hash.
func (e *EthAPI) GetTransactionByHash(hash common.Hash) (*rpc.EthTxJsonRPC, error) {
	methodName := "eth_getTransactionByHash"
	e.logger.Debug(methodName, "hash", hash.Hex())
	tx, err := e.backend.GetTransactionByHash(hash)
	logError(e.logger, err, methodName)
	return tx, err
}

// GetTransactionCount returns the account nonce for the given address at the specified block.
// This corresponds to the number of transactions sent from the address, including pending ones
// if blockNum == "pending". Returns 0 for non-existent accounts.
//
// ## Etheruem Nonce Behavior
//   - The nonce is a per-account counter.
//   - Is starts at 0 when the account is created and increments by 1 for each
//     successfully broadcasted transaction sent from that account.
//   - The nonce is NOT scoped per block but is global and persistent for each
//     account over time.
func (e *EthAPI) GetTransactionCount(
	address common.Address, blockNrOrHash rpc.BlockNumberOrHash,
) (*hexutil.Uint64, error) {
	methodName := "eth_getTransactionCount"
	e.logger.Debug(methodName, "address", address.Hex(), "block number or hash", blockNrOrHash)
	blockNum, err := e.backend.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}
	txCount, err := e.backend.GetTransactionCount(address, blockNum)
	logError(e.logger, err, methodName)
	return txCount, err
}

// GetTransactionReceipt returns the transaction receipt identified by hash. Note
// that a transaction that is successfully included in a block, even if it fails
// during execution (such as in the case of VM revert, out-of-gas, invalid
// opcode), will still produce a receipt
func (e *EthAPI) GetTransactionReceipt(
	hash common.Hash,
) (*TransactionReceipt, error) {
	methodName := "eth_getTransactionReceipt"
	e.logger.Debug(methodName, "hash", hash.Hex())
	out, err := e.backend.GetTransactionReceipt(hash)
	logError(e.logger, err, methodName)
	return out, err
}

// GetBlockTransactionCountByHash returns the number of transactions in the block identified by hash.
func (e *EthAPI) GetBlockTransactionCountByHash(
	blockHash common.Hash,
) (*hexutil.Uint, error) {
	methodName := "eth_getBlockTransactionCountByHash"
	e.logger.Debug(methodName, "hash", blockHash.Hex())
	txCount, err := e.backend.GetBlockTransactionCountByHash(blockHash)
	logError(e.logger, err, methodName)
	return txCount, err
}

// GetBlockTransactionCountByNumber returns the number of transactions in the
// block with the given block number.
func (e *EthAPI) GetBlockTransactionCountByNumber(
	blockNum rpc.BlockNumber,
) (*hexutil.Uint, error) {
	methodName := "eth_getBlockTransactionCountByNumber"
	e.logger.Debug(methodName, "height", blockNum.Int64())
	txCount, err := e.backend.GetBlockTransactionCountByNumber(blockNum)
	logError(e.logger, err, methodName)
	return txCount, err
}

// GetTransactionByBlockHashAndIndex returns the Ethereum-formatted transaction
// in the block with the given hash and specifed index in the block.
func (e *EthAPI) GetTransactionByBlockHashAndIndex(
	hash common.Hash, idx hexutil.Uint,
) (*rpc.EthTxJsonRPC, error) {
	methodName := "eth_getTransactionByBlockHashAndIndex"
	e.logger.Debug(methodName, "hash", hash.Hex(), "index", idx)
	out, err := e.backend.GetTransactionByBlockHashAndIndex(hash, idx)
	logError(e.logger, err, methodName)
	return out, err
}

// GetTransactionByBlockNumberAndIndex returns the Ethereum-formatted transaction
// in the block at the given block height and index within the block.
func (e *EthAPI) GetTransactionByBlockNumberAndIndex(
	blockNum rpc.BlockNumber, idx hexutil.Uint,
) (*rpc.EthTxJsonRPC, error) {
	methodName := "eth_getTransactionByBlockNumberAndIndex"
	e.logger.Debug(methodName, "number", blockNum, "index", idx)
	tx, err := e.backend.GetTransactionByBlockNumberAndIndex(blockNum, idx)
	logError(e.logger, err, methodName)
	return tx, err
}

// --------------------------------------------------------------------------
//                           Write Txs
// --------------------------------------------------------------------------

// SendRawTransaction send a raw Ethereum transaction.
// Allows developers to both send ETH from one address to another, write data
// on-chain, and interact with smart contracts.
func (e *EthAPI) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	methodName := "eth_sendRawTransaction"
	e.logger.Debug(methodName, "length", len(data))
	out, err := e.backend.SendRawTransaction(data)
	logError(e.logger, err, methodName)
	return out, err
}

// --------------------------------------------------------------------------
//                           Account Information
// --------------------------------------------------------------------------

// Accounts returns the list of accounts available to this node.
func (e *EthAPI) Accounts() ([]common.Address, error) {
	methodName := "eth_accounts"
	e.logger.Debug(methodName)
	accs, err := e.backend.Accounts()
	logError(e.logger, err, methodName)
	return accs, err
}

// GetBalance returns the provided account's balance up to the provided block number.
func (e *EthAPI) GetBalance(
	address common.Address, blockNrOrHash rpc.BlockNumberOrHash,
) (*hexutil.Big, error) {
	e.logger.Debug("eth_getBalance", "address", address.String(), "block number or hash", blockNrOrHash)
	return e.backend.GetBalance(address, blockNrOrHash)
}

// GetStorageAt returns the contract storage at the given address, block number, and key.
func (e *EthAPI) GetStorageAt(
	address common.Address, key string, blockNrOrHash rpc.BlockNumberOrHash,
) (hexutil.Bytes, error) {
	e.logger.Debug("eth_getStorageAt", "address", address.Hex(), "key", key, "block number or hash", blockNrOrHash)
	return e.backend.GetStorageAt(address, key, blockNrOrHash)
}

// GetCode returns the contract code at the given address and block number.
func (e *EthAPI) GetCode(
	address common.Address, blockNrOrHash rpc.BlockNumberOrHash,
) (hexutil.Bytes, error) {
	e.logger.Debug("eth_getCode", "address", address.Hex(), "block number or hash", blockNrOrHash)
	return e.backend.GetCode(address, blockNrOrHash)
}

// GetProof returns an account object with proof and any storage proofs
func (e *EthAPI) GetProof(address common.Address,
	storageKeys []string,
	blockNrOrHash rpc.BlockNumberOrHash,
) (*rpc.AccountResult, error) {
	e.logger.Debug("eth_getProof", "address", address.Hex(), "keys", storageKeys, "block number or hash", blockNrOrHash)
	return e.backend.GetProof(address, storageKeys, blockNrOrHash)
}

// --------------------------------------------------------------------------
//                           EVM/Smart Contract Execution
// --------------------------------------------------------------------------

// Call performs a raw contract call.
//
// Allows developers to read data from the blockchain which includes executing
// smart contracts. However, no data is published to the blockchain network.
func (e *EthAPI) Call(
	args evm.JsonTxArgs,
	blockNrOrHash rpc.BlockNumberOrHash,
	_ *rpc.StateOverride,
) (bz hexutil.Bytes, err error) {
	e.logger.Debug("eth_call", "args", args.String(), "block number or hash", blockNrOrHash)

	blockNum, err := e.backend.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		logError(e.logger, err, "eth_call")
		return bz, err
	}
	msgEthTxResp, err := e.backend.DoCall(args, blockNum)
	if err != nil {
		logError(e.logger, err, "eth_call")
		return bz, err
	}

	return (hexutil.Bytes)(msgEthTxResp.Ret), nil
}

// --------------------------------------------------------------------------
//                           Event Logs
// --------------------------------------------------------------------------
// FILTER API at ./filters/api.go

// --------------------------------------------------------------------------
//                           Chain Information
// --------------------------------------------------------------------------

// ProtocolVersion returns the supported Ethereum protocol version.
func (e *EthAPI) ProtocolVersion() hexutil.Uint {
	e.logger.Debug("eth_protocolVersion")
	return hexutil.Uint(eth.ProtocolVersion)
}

// GasPrice returns the current gas price based on Ethermint's gas price oracle.
func (e *EthAPI) GasPrice() (*hexutil.Big, error) {
	e.logger.Debug("eth_gasPrice")
	return e.backend.GasPrice()
}

// EstimateGas returns an estimate of gas usage for the given smart contract call.
func (e *EthAPI) EstimateGas(
	args evm.JsonTxArgs, blockNrOptional *rpc.BlockNumber,
) (hexutil.Uint64, error) {
	e.logger.Debug("eth_estimateGas")
	return e.backend.EstimateGas(args, blockNrOptional)
}

func (e *EthAPI) FeeHistory(blockCount gethmath.HexOrDecimal64,
	lastBlock gethrpc.BlockNumber,
	rewardPercentiles []float64,
) (*rpc.FeeHistoryResult, error) {
	e.logger.Debug("eth_feeHistory")
	return e.backend.FeeHistory(blockCount, lastBlock, rewardPercentiles)
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee
// transactions.
func (e *EthAPI) MaxPriorityFeePerGas() (*hexutil.Big, error) {
	e.logger.Debug("eth_maxPriorityFeePerGas")
	head, err := e.backend.CurrentHeader()
	if err != nil {
		return nil, err
	}
	tipcap, err := e.backend.SuggestGasTipCap(head.BaseFee)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(tipcap), nil
}

// ChainId is the EIP-155 replay-protection chain id for the current ethereum
// chain config.
func (e *EthAPI) ChainId() (*hexutil.Big, error) { //nolint
	e.logger.Debug("eth_chainId")
	return e.backend.ChainID(), nil
}

// --------------------------------------------------------------------------
//                           Uncles
// --------------------------------------------------------------------------

// GetUncleByBlockHashAndIndex returns the uncle identified by hash and index.
// Always returns nil.
func (e *EthAPI) GetUncleByBlockHashAndIndex(
	_ common.Hash, _ hexutil.Uint,
) map[string]any {
	return nil
}

// GetUncleByBlockNumberAndIndex returns the uncle identified by number and
// index. Always returns nil.
func (e *EthAPI) GetUncleByBlockNumberAndIndex(
	_, _ hexutil.Uint,
) map[string]any {
	return nil
}

// GetUncleCountByBlockHash returns the number of uncles in the block identified
// by hash. Always zero.
func (e *EthAPI) GetUncleCountByBlockHash(_ common.Hash) hexutil.Uint {
	return 0
}

// GetUncleCountByBlockNumber returns the number of uncles in the block
// identified by number. Always zero.
func (e *EthAPI) GetUncleCountByBlockNumber(_ rpc.BlockNumber) hexutil.Uint {
	return 0
}

// --------------------------------------------------------------------------
//                           Other
// --------------------------------------------------------------------------

// Syncing returns false in case the node is currently not syncing with the
// network. It can be up to date or has not yet received the latest block headers
// from its pears. In case it is synchronizing:
//
// - startingBlock: block number this node started to synchronize from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (e *EthAPI) Syncing() (any, error) {
	e.logger.Debug("eth_syncing")
	return e.backend.Syncing()
}

// GetTransactionLogs returns the logs given a transaction hash.
func (e *EthAPI) GetTransactionLogs(txHash common.Hash) ([]*gethcore.Log, error) {
	methodName := "eth_getTransactionLogs"
	e.logger.Debug(methodName, "hash", txHash)
	logs, err := e.backend.GetTransactionLogs(txHash)
	logError(e.logger, err, methodName)
	return logs, err
}

// FillTransaction fills the defaults (nonce, gas, gasPrice or 1559 fields)
// on a given unsigned transaction, and returns it to the caller for further
// processing (signing + broadcast).
func (e *EthAPI) FillTransaction(
	args evm.JsonTxArgs,
) (*rpc.SignTransactionResult, error) {
	e.logger.Debug("eth_fillTransaction")
	// Set some sanity defaults and terminate on failure
	args, err := e.backend.SetTxDefaults(args)
	if err != nil {
		return nil, err
	}

	// Assemble the transaction and obtain rlp
	tx := args.ToMsgEthTx().AsTransaction()

	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return &rpc.SignTransactionResult{
		Raw: data,
		Tx:  tx,
	}, nil
}

// GetPendingTransactions returns the transactions that are in the transaction
// pool and have a from address that is one of the accounts this node manages.
func (e *EthAPI) GetPendingTransactions() ([]*rpc.EthTxJsonRPC, error) {
	e.logger.Debug("eth_getPendingTransactions")

	txs, err := e.backend.PendingTransactions()
	if err != nil {
		return nil, err
	}

	result := make([]*rpc.EthTxJsonRPC, 0, len(txs))
	for _, tx := range txs {
		for _, msg := range (*tx).GetMsgs() {
			ethMsg, ok := msg.(*evm.MsgEthereumTx)
			if !ok {
				// not valid ethereum tx
				break
			}

			rpctx := rpc.NewRPCTxFromMsgEthTx(
				ethMsg,
				common.Hash{},
				uint64(0),
				uint64(0),
				nil,
				e.backend.ChainConfig().ChainID,
			)

			result = append(result, rpctx)
		}
	}

	return result, nil
}
