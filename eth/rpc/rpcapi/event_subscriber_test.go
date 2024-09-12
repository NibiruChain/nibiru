package rpcapi_test

import (
	"context"
	"os"
	"sync"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/suite"

	"github.com/cometbft/cometbft/libs/log"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/pubsub"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type Suite struct {
	suite.Suite
}

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
		t.Error("expect topic channel exist")
	}

	sub = rpcapi.MakeSubscription("2", event)
	es.Install <- sub
	<-sub.Installed
	newCh, ok := es.TopicChans[sub.Event]
	if !ok {
		t.Error("expect topic channel exist")
	}

	if newCh != ch {
		t.Error("expect topic channel unchanged")
	}
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

				err := deps.Ctx.EventManager().EmitTypedEvents(
					&evm.EventTransfer{},
					&evm.EventBlockBloom{
						Bloom: eth.BloomToString(bloom),
					},
				)
				s.NoError(err, "emitting bloom event failed")

				abciEvents := deps.Ctx.EventManager().ABCIEvents()

				bloomEvent := new(evm.EventBlockBloom)
				bloomEventType := gogoproto.MessageName(bloomEvent)

				err = testutil.AssertEventPresent(deps.Ctx.EventManager().Events(), bloomEventType)
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
