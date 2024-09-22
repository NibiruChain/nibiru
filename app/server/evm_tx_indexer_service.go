// Copyright (c) 2023-2024 Nibi, Inc.
package server

import (
	"context"
	"time"

	"github.com/cometbft/cometbft/libs/service"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cometbft/cometbft/types"

	"github.com/NibiruChain/nibiru/v2/eth/indexer"
)

const (
	ServiceName = "EVMTxIndexerService"

	NewBlockWaitTimeout = 60 * time.Second
)

// EVMTxIndexerService indexes transactions for json-rpc service.
type EVMTxIndexerService struct {
	service.BaseService

	txIndexer  *indexer.EVMTxIndexer
	client     rpcclient.Client
	cancelFunc context.CancelFunc
}

// NewEVMIndexerService returns a new service instance.
func NewEVMIndexerService(
	txIdxr *indexer.EVMTxIndexer,
	client rpcclient.Client,
) *EVMTxIndexerService {
	is := &EVMTxIndexerService{txIndexer: txIdxr, client: client}
	is.BaseService = *service.NewBaseService(nil, ServiceName, is)
	return is
}

// OnStart implements service.Service by subscribing for new blocks
// and indexing them by events.
func (service *EVMTxIndexerService) OnStart() error {
	ctx, cancel := context.WithCancel(context.Background())
	service.cancelFunc = cancel

	status, err := service.client.Status(ctx)
	if err != nil {
		return err
	}
	latestBlock := status.SyncInfo.LatestBlockHeight
	newBlockSignal := make(chan struct{}, 1)

	blockHeadersChan, err := service.client.Subscribe(
		ctx,
		ServiceName,
		types.QueryForEvent(types.EventNewBlockHeader).String(),
		0,
	)
	if err != nil {
		return err
	}

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done(): // Listen for context cancellation to stop the goroutine
				service.Logger.Info("Stopping indexer goroutine")
				err := service.txIndexer.CloseDBAndExit()
				if err != nil {
					service.Logger.Error("Error closing indexer DB", "err", err)
				}
				return
			case msg := <-blockHeadersChan:
				eventDataHeader := msg.Data.(types.EventDataNewBlockHeader)
				if eventDataHeader.Header.Height > latestBlock {
					latestBlock = eventDataHeader.Header.Height
					// notify
					select {
					case newBlockSignal <- struct{}{}:
					default:
					}
				}
			}
		}
	}(ctx)

	lastBlock, err := service.txIndexer.LastIndexedBlock()
	if err != nil {
		return err
	}
	if lastBlock == -1 {
		lastBlock = latestBlock
	}
	for {
		if latestBlock <= lastBlock {
			// nothing to index. wait for signal of new block
			select {
			case <-newBlockSignal:
			case <-time.After(NewBlockWaitTimeout):
			}
			continue
		}
		for i := lastBlock + 1; i <= latestBlock; i++ {
			block, err := service.client.Block(ctx, &i)
			if err != nil {
				service.Logger.Error("failed to fetch block", "height", i, "err", err)
				break
			}
			blockResult, err := service.client.BlockResults(ctx, &i)
			if err != nil {
				service.Logger.Error("failed to fetch block result", "height", i, "err", err)
				break
			}
			if err := service.txIndexer.IndexBlock(block.Block, blockResult.TxsResults); err != nil {
				service.Logger.Error("failed to index block", "height", i, "err", err)
			}
			lastBlock = blockResult.Height
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
