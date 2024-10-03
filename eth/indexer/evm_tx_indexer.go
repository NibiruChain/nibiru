// Copyright (c) 2023-2024 Nibi, Inc.
package indexer

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

const (
	KeyPrefixTxHash  = 1
	KeyPrefixTxIndex = 2

	// TxIndexKeyLength is the length of tx-index key
	TxIndexKeyLength = 1 + 8 + 8
)

var _ eth.EVMTxIndexer = &EVMTxIndexer{}

// EVMTxIndexer implements a eth tx indexer on a KV db.
type EVMTxIndexer struct {
	db        dbm.DB
	logger    log.Logger
	clientCtx client.Context
}

// NewEVMTxIndexer creates the EVMTxIndexer
func NewEVMTxIndexer(db dbm.DB, logger log.Logger, clientCtx client.Context) *EVMTxIndexer {
	return &EVMTxIndexer{db, logger, clientCtx}
}

// IndexBlock index all the eth txs in a block through the following steps:
// - Iterates over all the Txs in Block
// - Parses eth Tx infos from cosmos-sdk events for every TxResult
// - Iterates over all the messages of the Tx
// - Builds and stores indexer.TxResult based on parsed events for every message
func (indexer *EVMTxIndexer) IndexBlock(block *tmtypes.Block, txResults []*abci.ResponseDeliverTx) error {
	height := block.Header.Height

	batch := indexer.db.NewBatch()
	defer batch.Close()

	// record index of valid eth tx during the iteration
	var ethTxIndex int32
	for txIndex, tx := range block.Txs {
		result := txResults[txIndex]
		isValidEnough, reason := rpc.TxIsValidEnough(result)
		if !isValidEnough {
			indexer.logger.Debug(
				"Skipped indexing of tx",
				"reason", reason,
				"tm_tx_hash", eth.TmTxHashToString(tx.Hash()),
			)
			continue
		}

		tx, err := indexer.clientCtx.TxConfig.TxDecoder()(tx)
		if err != nil {
			indexer.logger.Error("Fail to decode tx", "err", err, "block", height, "txIndex", txIndex)
			continue
		}

		if !isEthTx(tx) {
			continue
		}

		txs, err := rpc.ParseTxResult(result, tx)
		if err != nil {
			indexer.logger.Error("Fail to parse event", "err", err, "block", height, "txIndex", txIndex)
			continue
		}

		var cumulativeGasUsed uint64
		for msgIndex, msg := range tx.GetMsgs() {
			ethMsg := msg.(*evm.MsgEthereumTx)
			txHash := common.HexToHash(ethMsg.Hash)

			txResult := eth.TxResult{
				Height:     height,
				TxIndex:    uint32(txIndex),
				MsgIndex:   uint32(msgIndex),
				EthTxIndex: ethTxIndex,
			}
			if result.Code != abci.CodeTypeOK {
				// exceeds block gas limit scenario, set gas used to gas limit because that's what's charged by ante handler.
				// some old versions don't emit any events, so workaround here directly.
				txResult.GasUsed = ethMsg.GetGas()
				txResult.Failed = true
			} else {
				parsedTx := txs.GetTxByMsgIndex(msgIndex)
				if parsedTx == nil {
					indexer.logger.Error("msg index not found in events", "msgIndex", msgIndex)
					continue
				}
				if parsedTx.EthTxIndex >= 0 && parsedTx.EthTxIndex != ethTxIndex {
					indexer.logger.Error(
						"eth tx index don't match",
						"expect", ethTxIndex,
						"found", parsedTx.EthTxIndex,
						"height", height,
					)
				}
				txResult.GasUsed = parsedTx.GasUsed
				txResult.Failed = parsedTx.Failed
			}

			cumulativeGasUsed += txResult.GasUsed
			txResult.CumulativeGasUsed = cumulativeGasUsed
			ethTxIndex++

			if err := saveTxResult(indexer.clientCtx.Codec, batch, txHash, &txResult); err != nil {
				return errorsmod.Wrapf(err, "IndexBlock %d", height)
			}
		}
	}
	if err := batch.Write(); err != nil {
		return errorsmod.Wrapf(err, "IndexBlock %d, write batch", block.Height)
	}
	return nil
}

// LastIndexedBlock returns the latest indexed block number, returns -1 if db is empty
func (indexer *EVMTxIndexer) LastIndexedBlock() (int64, error) {
	return LoadLastBlock(indexer.db)
}

// FirstIndexedBlock returns the first indexed block number, returns -1 if db is empty
func (indexer *EVMTxIndexer) FirstIndexedBlock() (int64, error) {
	return LoadFirstBlock(indexer.db)
}

// GetByTxHash finds eth tx by eth tx hash
func (indexer *EVMTxIndexer) GetByTxHash(hash common.Hash) (*eth.TxResult, error) {
	bz, err := indexer.db.Get(TxHashKey(hash))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetByTxHash %s", hash.Hex())
	}
	if len(bz) == 0 {
		return nil, fmt.Errorf("tx not found, hash: %s", hash.Hex())
	}
	var txKey eth.TxResult
	if err := indexer.clientCtx.Codec.Unmarshal(bz, &txKey); err != nil {
		return nil, errorsmod.Wrapf(err, "GetByTxHash %s", hash.Hex())
	}
	return &txKey, nil
}

// GetByBlockAndIndex finds eth tx by block number and eth tx index
func (indexer *EVMTxIndexer) GetByBlockAndIndex(blockNumber int64, txIndex int32) (*eth.TxResult, error) {
	bz, err := indexer.db.Get(TxIndexKey(blockNumber, txIndex))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetByBlockAndIndex %d %d", blockNumber, txIndex)
	}
	if len(bz) == 0 {
		return nil, fmt.Errorf("tx not found, block: %d, eth-index: %d", blockNumber, txIndex)
	}
	return indexer.GetByTxHash(common.BytesToHash(bz))
}

// TxHashKey returns the key for db entry: `tx hash -> tx result struct`
func TxHashKey(hash common.Hash) []byte {
	return append([]byte{KeyPrefixTxHash}, hash.Bytes()...)
}

// TxIndexKey returns the key for db entry: `(block number, tx index) -> tx hash`
func TxIndexKey(blockNumber int64, txIndex int32) []byte {
	bz1 := sdk.Uint64ToBigEndian(uint64(blockNumber))
	bz2 := sdk.Uint64ToBigEndian(uint64(txIndex))
	return append(append([]byte{KeyPrefixTxIndex}, bz1...), bz2...)
}

// LoadLastBlock returns the latest indexed block number, returns -1 if db is empty
func LoadLastBlock(db dbm.DB) (int64, error) {
	it, err := db.ReverseIterator([]byte{KeyPrefixTxIndex}, []byte{KeyPrefixTxIndex + 1})
	if err != nil {
		return 0, errorsmod.Wrap(err, "LoadLastBlock")
	}
	defer it.Close()
	if !it.Valid() {
		return -1, nil
	}
	return parseBlockNumberFromKey(it.Key())
}

// LoadFirstBlock loads the first indexed block, returns -1 if db is empty
func LoadFirstBlock(db dbm.DB) (int64, error) {
	it, err := db.Iterator([]byte{KeyPrefixTxIndex}, []byte{KeyPrefixTxIndex + 1})
	if err != nil {
		return 0, errorsmod.Wrap(err, "LoadFirstBlock")
	}
	defer it.Close()
	if !it.Valid() {
		return -1, nil
	}
	return parseBlockNumberFromKey(it.Key())
}

// CloseDBAndExit should be called upon stopping the indexer
func (indexer *EVMTxIndexer) CloseDBAndExit() error {
	indexer.logger.Info("Closing EVMTxIndexer DB")
	err := indexer.db.Close()
	if err != nil {
		return errorsmod.Wrap(err, "CloseDBAndExit")
	}
	return nil
}

// isEthTx check if the tx is an eth tx
func isEthTx(tx sdk.Tx) bool {
	extTx, ok := tx.(authante.HasExtensionOptionsTx)
	if !ok {
		return false
	}
	opts := extTx.GetExtensionOptions()
	if len(opts) != 1 || opts[0].GetTypeUrl() != "/eth.evm.v1.ExtensionOptionsEthereumTx" {
		return false
	}
	return true
}

// saveTxResult index the txResult into the kv db batch
func saveTxResult(codec codec.Codec, batch dbm.Batch, txHash common.Hash, txResult *eth.TxResult) error {
	bz := codec.MustMarshal(txResult)
	if err := batch.Set(TxHashKey(txHash), bz); err != nil {
		return errorsmod.Wrap(err, "set tx-hash key")
	}
	if err := batch.Set(TxIndexKey(txResult.Height, txResult.EthTxIndex), txHash.Bytes()); err != nil {
		return errorsmod.Wrap(err, "set tx-index key")
	}
	return nil
}

func parseBlockNumberFromKey(key []byte) (int64, error) {
	if len(key) != TxIndexKeyLength {
		return 0, fmt.Errorf("wrong tx index key length, expect: %d, got: %d", TxIndexKeyLength, len(key))
	}

	return int64(sdk.BigEndianToUint64(key[1:9])), nil
}
