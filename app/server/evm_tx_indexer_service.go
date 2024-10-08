// Copyright (c) 2023-2024 Nibi, Inc.
package server

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/cometbft/cometbft/libs/service"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cometbft/cometbft/types"

	"github.com/NibiruChain/nibiru/v2/eth/indexer"
)

const (
	EVMTxIndexerServiceName = "EVMTxIndexerService"

	NewBlockWaitTimeout = 60 * time.Second
)

// EVMTxIndexerService indexes transactions for json-rpc service.
type EVMTxIndexerService struct {
	service.BaseService

	evmTxIndexer *indexer.EVMTxIndexer
	rpcClient    rpcclient.Client
	cancelFunc   context.CancelFunc
}

// NewEVMIndexerService returns a new service instance.
func NewEVMIndexerService(evmTxIndexer *indexer.EVMTxIndexer, rpcClient rpcclient.Client) *EVMTxIndexerService {
	indexerService := &EVMTxIndexerService{evmTxIndexer: evmTxIndexer, rpcClient: rpcClient}
	indexerService.BaseService = *service.NewBaseService(nil, EVMTxIndexerServiceName, indexerService)
	return indexerService
}

// OnStart implements service.Service by subscribing for new blocks
// and indexing them by events.
func (service *EVMTxIndexerService) OnStart() error {
	ctx, cancel := context.WithCancel(context.Background())
	service.cancelFunc = cancel

	status, err := service.rpcClient.Status(ctx)
	if err != nil {
		return err
	}

	// chainHeightStorage is used within goroutine and the indexer loop so, using atomic for read/write
	var chainHeightStorage int64
	atomic.StoreInt64(&chainHeightStorage, status.SyncInfo.LatestBlockHeight)

	newBlockSignal := make(chan struct{}, 1)
	blockHeadersChan, err := service.rpcClient.Subscribe(
		ctx,
		EVMTxIndexerServiceName,
		types.QueryForEvent(types.EventNewBlockHeader).String(),
		0,
	)
	if err != nil {
		return err
	}

	// Goroutine listening for new blocks
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				service.Logger.Info("Stopping indexer goroutine")
				err := service.evmTxIndexer.CloseDBAndExit()
				if err != nil {
					service.Logger.Error("Error closing indexer DB", "err", err)
				}
				return
			case msg := <-blockHeadersChan:
				eventDataHeader := msg.Data.(types.EventDataNewBlockHeader)
				currentChainHeight := eventDataHeader.Header.Height
				chainHeight := atomic.LoadInt64(&chainHeightStorage)
				if currentChainHeight > chainHeight {
					atomic.StoreInt64(&chainHeightStorage, currentChainHeight)
					// notify
					select {
					case newBlockSignal <- struct{}{}:
					default:
					}
				}
			}
		}
	}(ctx)

	lastIndexedHeight, err := service.evmTxIndexer.LastIndexedBlock()
	if err != nil {
		return err
	}
	if lastIndexedHeight == -1 {
		lastIndexedHeight = atomic.LoadInt64(&chainHeightStorage)
	}

	// Indexer loop
	for {
		chainHeight := atomic.LoadInt64(&chainHeightStorage)
		if chainHeight <= lastIndexedHeight {
			// nothing to index. wait for signal of new block
			select {
			case <-newBlockSignal:
			case <-time.After(NewBlockWaitTimeout):
			}
			continue
		}
		for i := lastIndexedHeight + 1; i <= chainHeight; i++ {
			block, err := service.rpcClient.Block(ctx, &i)
			if err != nil {
				service.Logger.Error("failed to fetch block", "height", i, "err", err)
				break
			}
			blockResult, err := service.rpcClient.BlockResults(ctx, &i)
			if err != nil {
				service.Logger.Error("failed to fetch block result", "height", i, "err", err)
				break
			}
			if err := service.evmTxIndexer.IndexBlock(block.Block, blockResult.TxsResults); err != nil {
				service.Logger.Error("failed to index block", "height", i, "err", err)
			}
			lastIndexedHeight = blockResult.Height
		}
	}
}

func (service *EVMTxIndexerService) OnStop() {
	service.Logger.Info("Stopping EVMTxIndexerService")
	if service.cancelFunc != nil {
		service.Logger.Info("Calling EVMIndexerService CancelFunc")
		service.cancelFunc()
	}
}
