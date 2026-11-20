// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"
)

// ErrTxNotFound reports that the index has no record of a transaction hash.
// It means "not indexed", not "index unavailable": callers can look in the
// mempool next, but must not treat other index errors as a miss.
var ErrTxNotFound = errors.New("ethereum transaction not found")

// EVMTxIndexer defines the interface of custom eth tx indexer.
type EVMTxIndexer interface {
	// LastIndexedBlock returns -1 if indexer db is empty
	LastIndexedBlock() (int64, error)
	IndexBlock(*cmttypes.Block, []*abci.ResponseDeliverTx) error

	// GetByTxHash returns [ErrTxNotFound] if the transaction is absent.
	GetByTxHash(common.Hash) (*TxResult, error)
	// GetByBlockAndIndex returns nil if tx not found.
	GetByBlockAndIndex(int64, int32) (*TxResult, error)
}
