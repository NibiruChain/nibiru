// Copyright (c) 2023-2024 Nibi, Inc.
package backend

import (
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

// GetLogs returns all the logs from all the ethereum transactions in a block.
func (b *EVMBackend) GetLogs(hash common.Hash) ([][]*gethcore.Log, error) {
	resBlock, err := b.TendermintBlockByHash(hash)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		return nil, errors.Errorf("block not found for hash %s", hash)
	}
	return b.GetLogsByHeight(&resBlock.Block.Header.Height)
}

// GetLogsByHeight returns all the logs from all the ethereum transactions in a block.
func (b *EVMBackend) GetLogsByHeight(height *int64) ([][]*gethcore.Log, error) {
	// NOTE: we query the state in case the tx result logs are not persisted after an upgrade.
	blockRes, err := b.TendermintBlockResultByNumber(height)
	if err != nil {
		return nil, err
	}

	return GetLogsFromBlockResults(blockRes)
}

// BloomStatus returns:
//   - bloomBitsBlocks: The number of blocks a single bloom bit section vector
//     contains on the server side.
//   - bloomSections: The number of processed sections maintained by the indexer.
func (b *EVMBackend) BloomStatus() (
	bloomBitBlocks, bloomSections uint64,
) {
	return 4096, 0
}
