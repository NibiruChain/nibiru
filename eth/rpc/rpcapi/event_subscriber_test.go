package rpcapi_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	gogoproto "github.com/cosmos/gogoproto/proto"

	"github.com/cometbft/cometbft/libs/log"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/pubsub"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

func TestEventSubscriber(t *testing.T) {
	index := make(rpcapi.FilterIndex)
	for i := filters.UnknownSubscription; i < filters.LastIndexSubscription; i++ {
		index[i] = make(map[rpc.ID]*rpcapi.Subscription)
	}
	es := &rpcapi.EventSubscriber{
		Logger:     log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		Ctx:        context.Background(),
		LightMode:  false,
		Index:      index,
		TopicChans: make(map[string]chan<- coretypes.ResultEvent, len(index)),
		IndexMux:   new(sync.RWMutex),
		Install:    make(chan *rpcapi.Subscription),
		Uninstall:  make(chan *rpcapi.Subscription),
		EventBus:   pubsub.NewEventBus(),
	}
	go es.EventLoop()

	event := "event"
	sub := rpcapi.MakeSubscription("1", event)
	es.Install <- sub
	<-sub.Installed
	ch, ok := es.TopicChans[sub.Event]
	if !ok {
		t.Errorf("expect topic channel exist: event %v", sub.Event)
	}

	sub = rpcapi.MakeSubscription("2", event)
	es.Install <- sub
	<-sub.Installed
	newCh, ok := es.TopicChans[sub.Event]
	if !ok {
		t.Errorf("expect topic channel exist: event %v", sub.Event)
	}

	if newCh != ch {
		t.Errorf("expect topic channel unchanged: event %v", sub.Event)
	}
}

func TestEventSubscriberUninstallKeepsSharedTopic(t *testing.T) {
	es := newTestEventSubscriber()
	go es.EventLoop()

	event := "shared.event"
	sub1 := rpcapi.MakeSubscription("1", event)
	sub2 := rpcapi.MakeSubscription("2", event)
	es.Install <- sub1
	<-sub1.Installed
	ch := es.TopicChans[event]

	es.Install <- sub2
	<-sub2.Installed

	es.Uninstall <- sub1
	select {
	case _, ok := <-sub1.ErrCh:
		if ok {
			t.Fatalf("expected uninstalled subscription error channel to close")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for subscription uninstall")
	}

	if _, ok := es.Index[filters.LogsSubscription][sub1.Id]; ok {
		t.Fatalf("expected subscription %s to be removed from index", sub1.Id)
	}
	if _, ok := es.Index[filters.LogsSubscription][sub2.Id]; !ok {
		t.Fatalf("expected subscription %s to remain in index", sub2.Id)
	}
	if got := es.TopicChans[event]; got != ch {
		t.Fatalf("expected shared topic channel to remain installed")
	}
}

func TestEventSubscriberConsumeEventsPublishesMatchingTopic(t *testing.T) {
	tmWSClient := &rpcclient.WSClient{ResponsesCh: make(chan rpctypes.RPCResponse)}
	es := rpcapi.NewEventSubscriber(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		tmWSClient,
	)

	event := "consume.event"
	sub := rpcapi.MakeSubscription("1", event)
	es.Install <- sub
	<-sub.Installed

	eventCh, unsubscribe, err := es.EventBus.Subscribe(event)
	if err != nil {
		t.Fatalf("expected event bus subscription: %v", err)
	}
	defer unsubscribe()

	sendResultEvent(t, tmWSClient, coretypes.ResultEvent{})
	sendResultEvent(t, tmWSClient, coretypes.ResultEvent{Query: "unmatched.event"})

	want := coretypes.ResultEvent{Query: event}
	sendResultEvent(t, tmWSClient, want)

	select {
	case got := <-eventCh:
		if got.Query != want.Query {
			t.Fatalf("unexpected event query: got %q want %q", got.Query, want.Query)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for matching event")
	}
}

func sendResultEvent(t *testing.T, tmWSClient *rpcclient.WSClient, event coretypes.ResultEvent) {
	t.Helper()

	bz, err := tmjson.Marshal(event)
	if err != nil {
		t.Fatalf("marshal result event: %v", err)
	}
	tmWSClient.ResponsesCh <- rpctypes.RPCResponse{Result: bz}
}

func (s *Suite) TestParseBloomFromEvents() {
	for _, tc := range []struct {
		name           string
		endBlockEvents func() (gethcore.Bloom, []abci.Event)
		wantErr        string
	}{
		{
			name: "happy: empty events",
			endBlockEvents: func() (gethcore.Bloom, []abci.Event) {
				return *new(gethcore.Bloom), []abci.Event{}
			},
			wantErr: "",
		},
		{
			name: "happy: events with bloom included",
			endBlockEvents: func() (gethcore.Bloom, []abci.Event) {
				deps := evmtest.NewTestDeps()

				// populate valid bloom
				bloom := gethcore.Bloom{}
				dummyBz := []byte("dummybloom")
				copy(bloom[:], dummyBz)

				err := deps.Ctx().EventManager().EmitTypedEvents(
					&evm.EventTransfer{},
					&evm.EventBlockBloom{
						Bloom: eth.BloomToHex(bloom),
					},
				)
				s.NoError(err, "emitting bloom event failed")

				abciEvents := deps.Ctx().EventManager().ABCIEvents()

				bloomEvent := new(evm.EventBlockBloom)
				bloomEventType := gogoproto.MessageName(bloomEvent)

				err = testutil.AssertEventPresent(deps.Ctx().EventManager().Events(), bloomEventType)
				s.Require().NoError(err)

				return bloom, abciEvents
			},
			wantErr: "",
		},
	} {
		s.Run(tc.name, func() {
			wantBloom, events := tc.endBlockEvents()
			bloom, err := rpcapi.ParseBloomFromEvents(events)

			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}

			s.Require().Equal(wantBloom, bloom)
		})
	}
}

func newTestEventSubscriber() *rpcapi.EventSubscriber {
	index := make(rpcapi.FilterIndex)
	for i := filters.UnknownSubscription; i < filters.LastIndexSubscription; i++ {
		index[i] = make(map[rpc.ID]*rpcapi.Subscription)
	}

	return &rpcapi.EventSubscriber{
		Logger:     log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		Ctx:        context.Background(),
		LightMode:  false,
		Index:      index,
		TopicChans: make(map[string]chan<- coretypes.ResultEvent, len(index)),
		IndexMux:   new(sync.RWMutex),
		Install:    make(chan *rpcapi.Subscription),
		Uninstall:  make(chan *rpcapi.Subscription),
		EventBus:   pubsub.NewEventBus(),
	}
}
