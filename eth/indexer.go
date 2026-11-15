// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"
)

// ErrTxNotFound identifies a transaction absent from the index.
var ErrTxNotFound = errors.New("ethereum transaction not found")

// EVMTxIndexer defines the interface of custom eth tx indexer.
type EVMTxIndexer interface {
	// LastIndexedBlock returns -1 if indexer db is empty
	LastIndexedBlock() (int64, error)
	IndexBlock(*cmttypes.Block, []*abci.ResponseDeliverTx) error

	// GetByTxHash returns ErrTxNotFound if the transaction is absent.
	GetByTxHash(common.Hash) (*TxResult, error)
	// GetByBlockAndIndex returns nil if tx not found.
	GetByBlockAndIndex(int64, int32) (*TxResult, error)
}
