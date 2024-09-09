// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	rpcbackend "github.com/NibiruChain/nibiru/v2/eth/rpc/backend"

	"github.com/cometbft/cometbft/libs/log"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// FiltersAPI offers support to create and manage filters. This will allow
// external clients to retrieve various information related to the Ethereum
// protocol such as blocks, transactions and logs.
type FiltersAPI struct {
	logger    log.Logger
	clientCtx client.Context
	backend   *rpcbackend.Backend
	events    *EventSubscriber
	filtersMu sync.Mutex
	filters   map[gethrpc.ID]*filter
}

// consider a filter inactive if it has not been polled for within deadlineForInactivity
func deadlineForInactivity() time.Duration { return 5 * time.Minute }

// filter is a helper struct that holds meta information over the filter type and
// associated subscription in the event system.
type filter struct {
	typ      filters.Type
	deadline *time.Timer // filter is inactive when deadline triggers
	hashes   []common.Hash
	crit     filters.FilterCriteria
	logs     []*gethcore.Log
	s        *Subscription // associated subscription in event system
}

// NewImplFiltersAPI returns a new FiltersAPI instance.
func NewImplFiltersAPI(
	logger log.Logger,
	clientCtx client.Context,
	tmWSClient *rpcclient.WSClient,
	backend *rpcbackend.Backend,
) *FiltersAPI {
	logger = logger.With("api", "filter")
	api := &FiltersAPI{
		logger:    logger,
		clientCtx: clientCtx,
		backend:   backend,
		filters:   make(map[gethrpc.ID]*filter),
		events:    NewEventSubscriber(logger, tmWSClient),
	}

	go api.timeoutLoop()

	return api
}

// timeoutLoop runs every 5 minutes and deletes filters that have not been recently used.
// Tt is started when the api is created.
func (api *FiltersAPI) timeoutLoop() {
	ticker := time.NewTicker(deadlineForInactivity())
	defer ticker.Stop()

	for {
		<-ticker.C
		api.filtersMu.Lock()
		// #nosec G705
		for id, f := range api.filters {
			select {
			case <-f.deadline.C:
				f.s.Unsubscribe(api.events)
				delete(api.filters, id)
			default:
				continue
			}
		}
		api.filtersMu.Unlock()
	}
}

// NewPendingTransactionFilter creates a filter that fetches pending transaction
// hashes as transactions enter the pending state.
//
// It is part of the filter package because this filter can be used through the
// `eth_getFilterChanges` polling method that is also used for log filters.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newPendingTransactionFilter
func (api *FiltersAPI) NewPendingTransactionFilter() gethrpc.ID {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if len(api.filters) >= int(api.backend.RPCFilterCap()) {
		return gethrpc.ID("error creating pending tx filter: max limit reached")
	}

	pendingTxSub, cancelSubs, err := api.events.SubscribePendingTxs()
	if err != nil {
		// wrap error on the ID
		return gethrpc.ID(fmt.Sprintf("error creating pending tx filter: %s", err.Error()))
	}

	api.filters[pendingTxSub.ID()] = &filter{
		typ:      filters.PendingTransactionsSubscription,
		deadline: time.NewTimer(deadlineForInactivity()),
		hashes:   make([]common.Hash, 0),
		s:        pendingTxSub,
	}

	go func(txsCh <-chan coretypes.ResultEvent, errCh <-chan error) {
		defer cancelSubs()

		for {
			select {
			case ev, ok := <-txsCh:
				if !ok {
					api.filtersMu.Lock()
					delete(api.filters, pendingTxSub.ID())
					api.filtersMu.Unlock()
					return
				}

				data, ok := ev.Data.(tmtypes.EventDataTx)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", ev.Data))
					continue
				}

				tx, err := api.clientCtx.TxConfig.TxDecoder()(data.Tx)
				if err != nil {
					api.logger.Debug("fail to decode tx", "error", err.Error())
					continue
				}

				api.filtersMu.Lock()
				if f, found := api.filters[pendingTxSub.ID()]; found {
					for _, msg := range tx.GetMsgs() {
						ethTx, ok := msg.(*evm.MsgEthereumTx)
						if ok {
							f.hashes = append(f.hashes, ethTx.AsTransaction().Hash())
						}
					}
				}
				api.filtersMu.Unlock()
			case <-errCh:
				api.filtersMu.Lock()
				delete(api.filters, pendingTxSub.ID())
				api.filtersMu.Unlock()
			}
		}
	}(pendingTxSub.EventCh, pendingTxSub.Error())

	return pendingTxSub.ID()
}

// NewPendingTransactions creates a subscription that is triggered each time a
// transaction enters the transaction pool and was signed from one of the
// transactions this nodes manages.
func (api *FiltersAPI) NewPendingTransactions(ctx context.Context) (*gethrpc.Subscription, error) {
	notifier, supported := gethrpc.NotifierFromContext(ctx)
	if !supported {
		return &gethrpc.Subscription{}, gethrpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	ctx, cancelFn := context.WithTimeout(context.Background(), deadlineForInactivity())
	defer cancelFn()

	api.events.WithContext(ctx)

	pendingTxSub, cancelSubs, err := api.events.SubscribePendingTxs()
	if err != nil {
		return nil, err
	}

	go func(txsCh <-chan coretypes.ResultEvent) {
		defer cancelSubs()

		for {
			select {
			case ev, ok := <-txsCh:
				if !ok {
					api.filtersMu.Lock()
					delete(api.filters, pendingTxSub.ID())
					api.filtersMu.Unlock()
					return
				}

				data, ok := ev.Data.(tmtypes.EventDataTx)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", ev.Data))
					continue
				}

				tx, err := api.clientCtx.TxConfig.TxDecoder()(data.Tx)
				if err != nil {
					api.logger.Debug("fail to decode tx", "error", err.Error())
					continue
				}

				for _, msg := range tx.GetMsgs() {
					ethTx, ok := msg.(*evm.MsgEthereumTx)
					if ok {
						_ = notifier.Notify(rpcSub.ID, ethTx.AsTransaction().Hash()) // #nosec G703
					}
				}
			case <-rpcSub.Err():
				pendingTxSub.Unsubscribe(api.events)
				return
			case <-notifier.Closed():
				pendingTxSub.Unsubscribe(api.events)
				return
			}
		}
	}(pendingTxSub.EventCh)

	return rpcSub, err
}

// NewBlockFilter creates a filter that fetches blocks that are imported into the
// chain. It is part of the filter package since polling goes with
// eth_getFilterChanges.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (api *FiltersAPI) NewBlockFilter() gethrpc.ID {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if len(api.filters) >= int(api.backend.RPCFilterCap()) {
		return gethrpc.ID("error creating block filter: max limit reached")
	}

	headerSub, cancelSubs, err := api.events.SubscribeNewHeads()
	if err != nil {
		// wrap error on the ID
		return gethrpc.ID(fmt.Sprintf("error creating block filter: %s", err.Error()))
	}

	api.filters[headerSub.ID()] = &filter{typ: filters.BlocksSubscription, deadline: time.NewTimer(deadlineForInactivity()), hashes: []common.Hash{}, s: headerSub}

	go func(headersCh <-chan coretypes.ResultEvent, errCh <-chan error) {
		defer cancelSubs()

		for {
			select {
			case ev, ok := <-headersCh:
				if !ok {
					api.filtersMu.Lock()
					delete(api.filters, headerSub.ID())
					api.filtersMu.Unlock()
					return
				}

				data, ok := ev.Data.(tmtypes.EventDataNewBlockHeader)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", ev.Data))
					continue
				}

				api.filtersMu.Lock()
				if f, found := api.filters[headerSub.ID()]; found {
					f.hashes = append(f.hashes, common.BytesToHash(data.Header.Hash()))
				}
				api.filtersMu.Unlock()
			case <-errCh:
				api.filtersMu.Lock()
				delete(api.filters, headerSub.ID())
				api.filtersMu.Unlock()
				return
			}
		}
	}(headerSub.EventCh, headerSub.Error())

	return headerSub.ID()
}

// NewHeads send a notification each time a new block (and thus, block header) is
// added to the chain.
func (api *FiltersAPI) NewHeads(ctx context.Context) (*gethrpc.Subscription, error) {
	notifier, supported := gethrpc.NotifierFromContext(ctx)
	if !supported {
		return &gethrpc.Subscription{}, gethrpc.ErrNotificationsUnsupported
	}

	api.events.WithContext(ctx)
	rpcSub := notifier.CreateSubscription()

	headersSub, cancelSubs, err := api.events.SubscribeNewHeads()
	if err != nil {
		return &gethrpc.Subscription{}, err
	}

	// Start go routine to continue executing without blocking while the
	// goroutine handles incoming events.
	// The routine receives a channel that receives block header events as an
	// argument.
	go func(headersCh <-chan coretypes.ResultEvent) {
		// This defer statement ensures the subscription is cancelled
		// when the goroutine exits.
		defer cancelSubs()

		// Listen for block header events and handle them based on the
		// type of event received.
		for {
			select {
			case ev, ok := <-headersCh:
				if !ok {
					headersSub.Unsubscribe(api.events)
					return
				}

				data, ok := ev.Data.(tmtypes.EventDataNewBlockHeader)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", ev.Data))
					continue
				}

				var baseFee *big.Int = nil
				bloom, err := ParseBloomFromEvents(data.ResultEndBlock.Events)
				if err != nil {
					api.logger.Error("failed to parse bloom from end block events")
					return
				}
				header := rpc.EthHeaderFromTendermint(data.Header, bloom, baseFee)
				_ = notifier.Notify(rpcSub.ID, header) // #nosec G703
			case <-rpcSub.Err():
				headersSub.Unsubscribe(api.events)
				return
			case <-notifier.Closed():
				headersSub.Unsubscribe(api.events)
				return
			}
		}
	}(headersSub.EventCh)

	return rpcSub, err
}

// Logs creates a subscription that fires for all new log that match the given
// filter criteria.
// Implements "eth_logs".
func (api *FiltersAPI) Logs(
	ctx context.Context, crit filters.FilterCriteria,
) (*gethrpc.Subscription, error) {
	notifier, supported := gethrpc.NotifierFromContext(ctx)
	if !supported {
		return &gethrpc.Subscription{}, gethrpc.ErrNotificationsUnsupported
	}

	api.events.WithContext(ctx)
	rpcSub := notifier.CreateSubscription()

	logsSub, cancelSubs, err := api.events.SubscribeLogs(crit)
	if err != nil {
		return &gethrpc.Subscription{}, err
	}

	go func(logsCh <-chan coretypes.ResultEvent) {
		defer cancelSubs()

		for {
			select {
			case ev, ok := <-logsCh:
				if !ok {
					logsSub.Unsubscribe(api.events)
					return
				}

				// filter only events from EVM module txs
				_, isMsgEthereumTx := ev.Events[evm.TypeMsgEthereumTx]

				if !isMsgEthereumTx {
					// ignore transaction as it's not from the evm module
					return
				}

				// get transaction result data
				dataTx, ok := ev.Data.(tmtypes.EventDataTx)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", ev.Data))
					continue
				}

				txResponse, err := evm.DecodeTxResponse(dataTx.TxResult.Result.Data)
				if err != nil {
					api.logger.Error("fail to decode tx response", "error", err)
					return
				}

				logs := FilterLogs(evm.LogsToEthereum(txResponse.Logs), crit.FromBlock, crit.ToBlock, crit.Addresses, crit.Topics)

				for _, log := range logs {
					_ = notifier.Notify(rpcSub.ID, log) // #nosec G703
				}
			case <-rpcSub.Err(): // client send an unsubscribe request
				logsSub.Unsubscribe(api.events)
				return
			case <-notifier.Closed(): // connection dropped
				logsSub.Unsubscribe(api.events)
				return
			}
		}
	}(logsSub.EventCh)

	return rpcSub, err
}

// NewFilter creates a new filter and returns the filter id. It can be
// used to retrieve logs when the state changes. This method cannot be
// used to fetch logs that are already stored in the state.
//
// Default criteria for the from and to block are "latest".
// Using "latest" as block number will return logs for mined blocks.
// Using "pending" as block number returns logs for not yet mined (pending) blocks.
// In case logs are removed (chain reorg) previously returned logs are returned
// again but with the removed property set to true.
//
// In case "fromBlock" > "toBlock" an error is returned.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (api *FiltersAPI) NewFilter(criteria filters.FilterCriteria) (gethrpc.ID, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if len(api.filters) >= int(api.backend.RPCFilterCap()) {
		return gethrpc.ID(""), fmt.Errorf("error creating filter: max limit reached")
	}

	var (
		filterID = gethrpc.ID("")
		err      error
	)

	logsSub, cancelSubs, err := api.events.SubscribeLogs(criteria)
	if err != nil {
		return gethrpc.ID(""), err
	}

	filterID = logsSub.ID()

	api.filters[filterID] = &filter{
		typ:      filters.LogsSubscription,
		crit:     criteria,
		deadline: time.NewTimer(deadlineForInactivity()),
		hashes:   []common.Hash{},
		s:        logsSub,
	}

	go func(eventCh <-chan coretypes.ResultEvent) {
		defer cancelSubs()

		for {
			select {
			case ev, ok := <-eventCh:
				if !ok {
					api.filtersMu.Lock()
					delete(api.filters, filterID)
					api.filtersMu.Unlock()
					return
				}
				dataTx, ok := ev.Data.(tmtypes.EventDataTx)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", ev.Data))
					continue
				}

				txResponse, err := evm.DecodeTxResponse(dataTx.TxResult.Result.Data)
				if err != nil {
					api.logger.Error("fail to decode tx response", "error", err)
					return
				}

				logs := FilterLogs(evm.LogsToEthereum(txResponse.Logs), criteria.FromBlock, criteria.ToBlock, criteria.Addresses, criteria.Topics)

				api.filtersMu.Lock()
				if f, found := api.filters[filterID]; found {
					f.logs = append(f.logs, logs...)
				}
				api.filtersMu.Unlock()
			case <-logsSub.Error():
				api.filtersMu.Lock()
				delete(api.filters, filterID)
				api.filtersMu.Unlock()
				return
			}
		}
	}(logsSub.EventCh)

	return filterID, err
}

// GetLogs returns logs matching the given argument that are stored within the state.
// This function implements the "eth_getLogs" JSON-RPC service method.
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (api *FiltersAPI) GetLogs(
	ctx context.Context, crit filters.FilterCriteria,
) ([]*gethcore.Log, error) {

	var filter *Filter
	if crit.BlockHash != nil {
		// Block filter requested, construct a single-shot filter
		filter = NewBlockFilter(api.logger, *api.backend, crit)
	} else {
		// Convert the RPC block numbers into internal representations
		begin := gethrpc.LatestBlockNumber.Int64()
		if crit.FromBlock != nil {
			begin = crit.FromBlock.Int64()
		}
		end := gethrpc.LatestBlockNumber.Int64()
		if crit.ToBlock != nil {
			end = crit.ToBlock.Int64()
		}
		// Construct the range filter
		filter = NewRangeFilter(api.logger, *api.backend, begin, end, crit.Addresses, crit.Topics)
	}

	api.logger.Debug("eth_getLogs",
		"from_block", filter.criteria.FromBlock.String(),
		"to_block", filter.criteria.ToBlock.String(),
		"time", time.Now().UTC(),
	)

	// Run the filter and return all the logs
	logs, err := filter.Logs(ctx, int(api.backend.RPCLogsCap()), int64(api.backend.RPCBlockRangeCap()))
	if err != nil {
		return nil, err
	}

	return returnLogs(logs), err
}

// UninstallFilter removes the filter with the given filter id.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_uninstallfilter
func (api *FiltersAPI) UninstallFilter(id gethrpc.ID) bool {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	if found {
		delete(api.filters, id)
	}
	api.filtersMu.Unlock()

	if !found {
		return false
	}
	f.s.Unsubscribe(api.events)
	return true
}

// GetFilterLogs returns the logs for the filter with the given id.
// If the filter could not be found an empty array of logs is returned.
//
// This function implements the "eth_getFilterLogs" JSON-RPC service method.
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterlogs
func (api *FiltersAPI) GetFilterLogs(ctx context.Context, id gethrpc.ID) ([]*gethcore.Log, error) {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	api.filtersMu.Unlock()

	if !found {
		return returnLogs(nil), fmt.Errorf("filter %s not found", id)
	}

	if f.typ != filters.LogsSubscription {
		return returnLogs(nil), fmt.Errorf("filter %s doesn't have a LogsSubscription type: got %d", id, f.typ)
	}

	var filter *Filter
	if f.crit.BlockHash != nil {
		// Block filter requested, construct a single-shot filter
		filter = NewBlockFilter(api.logger, *api.backend, f.crit)
	} else {
		// Convert the RPC block numbers into internal representations
		begin := gethrpc.LatestBlockNumber.Int64()
		if f.crit.FromBlock != nil {
			begin = f.crit.FromBlock.Int64()
		}
		end := gethrpc.LatestBlockNumber.Int64()
		if f.crit.ToBlock != nil {
			end = f.crit.ToBlock.Int64()
		}
		// Construct the range filter
		filter = NewRangeFilter(api.logger, *api.backend, begin, end, f.crit.Addresses, f.crit.Topics)
	}
	// Run the filter and return all the logs
	logs, err := filter.Logs(ctx, int(api.backend.RPCLogsCap()), int64(api.backend.RPCBlockRangeCap()))
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), nil
}

// GetFilterChanges returns the logs for the filter with the given id since last
// time it was called. This can be used for polling.
//
// For pending transaction and block filters the result is []common.Hash.
// (pending)Log filters return []Log.
//
// This function implements the "eth_getFilterChanges" JSON-RPC service method.
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (api *FiltersAPI) GetFilterChanges(id gethrpc.ID) (interface{}, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	f, found := api.filters[id]
	if !found {
		return nil, fmt.Errorf("filter %s not found", id)
	}

	if !f.deadline.Stop() {
		// timer expired but filter is not yet removed in timeout loop
		// receive timer value and reset timer
		<-f.deadline.C
	}
	f.deadline.Reset(deadlineForInactivity())

	switch f.typ {
	case filters.PendingTransactionsSubscription, filters.BlocksSubscription:
		hashes := f.hashes
		f.hashes = nil
		return returnHashes(hashes), nil
	case filters.LogsSubscription, filters.MinedAndPendingLogsSubscription:
		logs := make([]*gethcore.Log, len(f.logs))
		copy(logs, f.logs)
		f.logs = []*gethcore.Log{}
		return returnLogs(logs), nil
	default:
		return nil, fmt.Errorf("invalid filter %s type %d", id, f.typ)
	}
}
