// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	tmquery "github.com/cometbft/cometbft/libs/pubsub/query"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/eth/rpc/pubsub"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var (
	txEventsQuery  = tmtypes.QueryForEvent(tmtypes.EventTx).String()
	evmEventsQuery = tmquery.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
		tmtypes.EventTypeKey,
		tmtypes.EventTx,
		sdk.EventTypeMessage,
		sdk.AttributeKeyModule, evm.ModuleName)).String()
	headerEventsQuery = tmtypes.QueryForEvent(tmtypes.EventNewBlockHeader).String()
)

// EventSubscriber creates subscriptions, processes events and broadcasts them to the
// subscription which match the subscription criteria using the Tendermint's RPC
// client.
type EventSubscriber struct {
	Logger     log.Logger
	Ctx        context.Context
	TmWSClient *rpcclient.WSClient

	// light client mode
	LightMode bool

	Index      FilterIndex
	TopicChans map[string]chan<- coretypes.ResultEvent
	IndexMux   *sync.RWMutex

	// Channels
	Install   chan *Subscription // install filter for event notification
	Uninstall chan *Subscription // remove filter for event notification
	EventBus  pubsub.EventBus
}

// NewEventSubscriber creates a new manager that listens for event on the given mux,
// parses and filters them. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewEventSubscriber(
	logger log.Logger,
	tmWSClient *rpcclient.WSClient,
) *EventSubscriber {
	index := make(FilterIndex)
	for i := filters.UnknownSubscription; i < filters.LastIndexSubscription; i++ {
		index[i] = make(map[gethrpc.ID]*Subscription)
	}

	es := &EventSubscriber{
		Logger:     logger,
		Ctx:        context.Background(),
		TmWSClient: tmWSClient,
		LightMode:  false,
		Index:      index,
		TopicChans: make(map[string]chan<- coretypes.ResultEvent, len(index)),
		IndexMux:   new(sync.RWMutex),
		Install:    make(chan *Subscription),
		Uninstall:  make(chan *Subscription),
		EventBus:   pubsub.NewEventBus(),
	}

	go es.EventLoop()
	go es.consumeEvents()
	return es
}

// WithContext sets a new context to the EventSystem. This is required to set a timeout context when
// a new filter is intantiated.
func (es *EventSubscriber) WithContext(ctx context.Context) {
	es.Ctx = ctx
}

// subscribe performs a new event subscription to a given Tendermint event.
// The subscription creates a unidirectional receive event channel to receive the ResultEvent.
func (es *EventSubscriber) subscribe(sub *Subscription) (*Subscription, pubsub.UnsubscribeFunc, error) {
	var (
		err      error
		cancelFn context.CancelFunc
	)

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	existingSubs := es.EventBus.Topics()
	for _, topic := range existingSubs {
		if topic == sub.Event {
			eventCh, unsubFn, err := es.EventBus.Subscribe(sub.Event)
			if err != nil {
				err := errors.Wrapf(err, "failed to subscribe to topic: %s", sub.Event)
				return nil, nil, err
			}

			sub.EventCh = eventCh
			return sub, unsubFn, nil
		}
	}

	switch sub.Typ {
	case filters.LogsSubscription:
		err = es.TmWSClient.Subscribe(ctx, sub.Event)
	case filters.BlocksSubscription:
		err = es.TmWSClient.Subscribe(ctx, sub.Event)
	case filters.PendingTransactionsSubscription:
		err = es.TmWSClient.Subscribe(ctx, sub.Event)
	default:
		err = fmt.Errorf("invalid filter subscription type %d", sub.Typ)
	}

	if err != nil {
		sub.ErrCh <- err
		return nil, nil, err
	}

	// wrap events in a go routine to prevent blocking
	es.Install <- sub
	<-sub.Installed

	eventCh, unsubFn, err := es.EventBus.Subscribe(sub.Event)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to subscribe to topic after installed: %s", sub.Event)
	}

	sub.EventCh = eventCh
	return sub, unsubFn, nil
}

// SubscribeLogs creates a subscription that will write all logs matching the
// given criteria to the given logs channel. Default value for the from and to
// block is "latest". If the fromBlock > toBlock an error is returned.
func (es *EventSubscriber) SubscribeLogs(crit filters.FilterCriteria) (*Subscription, pubsub.UnsubscribeFunc, error) {
	var from, to gethrpc.BlockNumber
	if crit.FromBlock == nil {
		from = gethrpc.LatestBlockNumber
	} else {
		from = gethrpc.BlockNumber(crit.FromBlock.Int64())
	}
	if crit.ToBlock == nil {
		to = gethrpc.LatestBlockNumber
	} else {
		to = gethrpc.BlockNumber(crit.ToBlock.Int64())
	}

	switch {
	// only interested in new mined logs, mined logs within a specific block range, or
	// logs from a specific block number to new mined blocks
	case (from == gethrpc.LatestBlockNumber && to == gethrpc.LatestBlockNumber),
		(from >= 0 && to >= 0 && to >= from),
		(from >= 0 && to == gethrpc.LatestBlockNumber):

		// Create a subscription that will write all logs matching the
		// given criteria to the given logs channel.
		sub := &Subscription{
			Id:        gethrpc.NewID(),
			Typ:       filters.LogsSubscription,
			Event:     evmEventsQuery,
			logsCrit:  crit,
			Created:   time.Now().UTC(),
			Logs:      make(chan []*gethcore.Log),
			Installed: make(chan struct{}, 1),
			ErrCh:     make(chan error, 1),
		}
		return es.subscribe(sub)

	default:
		return nil, nil, fmt.Errorf("invalid from and to block combination: from > to (%d > %d)", from, to)
	}
}

// SubscribeNewHeads subscribes to new block headers events.
func (es EventSubscriber) SubscribeNewHeads() (*Subscription, pubsub.UnsubscribeFunc, error) {
	sub := &Subscription{
		Id:        gethrpc.NewID(),
		Typ:       filters.BlocksSubscription,
		Event:     headerEventsQuery,
		Created:   time.Now().UTC(),
		Headers:   make(chan *gethcore.Header),
		Installed: make(chan struct{}, 1),
		ErrCh:     make(chan error, 1),
	}
	return es.subscribe(sub)
}

// SubscribePendingTxs subscribes to new pending transactions events from the mempool.
func (es EventSubscriber) SubscribePendingTxs() (*Subscription, pubsub.UnsubscribeFunc, error) {
	sub := &Subscription{
		Id:        gethrpc.NewID(),
		Typ:       filters.PendingTransactionsSubscription,
		Event:     txEventsQuery,
		Created:   time.Now().UTC(),
		Hashes:    make(chan []common.Hash),
		Installed: make(chan struct{}, 1),
		ErrCh:     make(chan error, 1),
	}
	return es.subscribe(sub)
}

type FilterIndex map[filters.Type]map[gethrpc.ID]*Subscription

// EventLoop (un)installs filters and processes mux events.
func (es *EventSubscriber) EventLoop() {
	for {
		select {
		case f := <-es.Install:
			es.IndexMux.Lock()
			es.Index[f.Typ][f.Id] = f
			ch := make(chan coretypes.ResultEvent)
			if err := es.EventBus.AddTopic(f.Event, ch); err != nil {
				es.Logger.Error("failed to add event topic to event bus", "topic", f.Event, "error", err.Error())
			} else {
				es.TopicChans[f.Event] = ch
			}
			es.IndexMux.Unlock()
			close(f.Installed)
		case f := <-es.Uninstall:
			es.IndexMux.Lock()
			delete(es.Index[f.Typ], f.Id)

			var channelInUse bool
			// #nosec G705
			for _, sub := range es.Index[f.Typ] {
				if sub.Event == f.Event {
					channelInUse = true
					break
				}
			}

			// remove topic only when channel is not used by other subscriptions
			if !channelInUse {
				if err := es.TmWSClient.Unsubscribe(es.Ctx, f.Event); err != nil {
					es.Logger.Error("failed to unsubscribe from query", "query", f.Event, "error", err.Error())
				}

				ch, ok := es.TopicChans[f.Event]
				if ok {
					es.EventBus.RemoveTopic(f.Event)
					close(ch)
					delete(es.TopicChans, f.Event)
				}
			}

			es.IndexMux.Unlock()
			close(f.ErrCh)
		}
	}
}

func (es *EventSubscriber) consumeEvents() {
	for {
		for rpcResp := range es.TmWSClient.ResponsesCh {
			var ev coretypes.ResultEvent

			if rpcResp.Error != nil {
				time.Sleep(5 * time.Second)
				continue
			} else if err := tmjson.Unmarshal(rpcResp.Result, &ev); err != nil {
				es.Logger.Error("failed to JSON unmarshal ResponsesCh result event", "error", err.Error())
				continue
			}

			if len(ev.Query) == 0 {
				// skip empty responses
				continue
			}

			es.IndexMux.RLock()
			ch, ok := es.TopicChans[ev.Query]
			es.IndexMux.RUnlock()
			if !ok {
				es.Logger.Debug("channel for subscription not found", "topic", ev.Query)
				es.Logger.Debug("list of available channels", "channels", es.EventBus.Topics())
				continue
			}

			// gracefully handle lagging subscribers
			t := time.NewTimer(time.Second)
			select {
			case <-t.C:
				es.Logger.Debug("dropped event during lagging subscription", "topic", ev.Query)
			case ch <- ev:
			}
		}

		time.Sleep(time.Second)
	}
}

func MakeSubscription(id, event string) *Subscription {
	return &Subscription{
		Id:        gethrpc.ID(id),
		Typ:       filters.LogsSubscription,
		Event:     event,
		Created:   time.Now(),
		Logs:      make(chan []*gethcore.Log),
		Hashes:    make(chan []common.Hash),
		Headers:   make(chan *gethcore.Header),
		Installed: make(chan struct{}),
		EventCh:   make(chan coretypes.ResultEvent),
		ErrCh:     make(chan error),
	}
}
