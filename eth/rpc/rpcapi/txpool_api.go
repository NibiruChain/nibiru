// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"github.com/cometbft/cometbft/libs/log"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/NibiruChain/nibiru/eth/rpc"
)

// TxPoolAPI offers and API for the transaction pool. It only operates on data
// that is non-confidential.
type TxPoolAPI struct {
	logger log.Logger
}

// NewImplTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewImplTxPoolAPI(logger log.Logger) *TxPoolAPI {
	return &TxPoolAPI{
		logger: logger.With("module", "txpool"),
	}
}

// Content returns the transactions contained within the transaction pool
func (api *TxPoolAPI) Content() (
	map[string]map[string]map[string]*rpc.EthTxJsonRPC, error,
) {
	api.logger.Debug("txpool_content")
	content := map[string]map[string]map[string]*rpc.EthTxJsonRPC{
		"pending": make(map[string]map[string]*rpc.EthTxJsonRPC),
		"queued":  make(map[string]map[string]*rpc.EthTxJsonRPC),
	}
	return content, nil
}

// Inspect returns the content of the transaction pool and flattens it into an
func (api *TxPoolAPI) Inspect() (map[string]map[string]map[string]string, error) {
	api.logger.Debug("txpool_inspect")
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	return content, nil
}

// Status returns the number of pending and queued transaction in the pool.
func (api *TxPoolAPI) Status() map[string]hexutil.Uint {
	api.logger.Debug("txpool_status")
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(0),
		"queued":  hexutil.Uint(0),
	}
}
