// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"
)

// Subscription defines a wrapper for the private subscription
type Subscription struct {
	Id        rpc.ID
	Typ       filters.Type
	Event     string
	Created   time.Time
	logsCrit  filters.FilterCriteria
	Logs      chan []*gethcore.Log
	Hashes    chan []common.Hash
	Headers   chan *gethcore.Header
	Installed chan struct{} // closed when the filter is installed
	// Consensus result event channel
	EventCh <-chan coretypes.ResultEvent
	ErrCh   chan error
}

// ID returns the underlying subscription RPC identifier.
func (s Subscription) ID() rpc.ID {
	return s.Id
}

// Unsubscribe from the current subscription to Tendermint Websocket. It sends an error to the
// subscription error channel if unsubscribe fails.
func (s *Subscription) Unsubscribe(es *EventSubscriber) {
	go func() {
	uninstallLoop:
		for {
			// write uninstall request and consume logs/hashes. This prevents
			// the eventLoop broadcast method to deadlock when writing to the
			// filter event channel while the subscription loop is waiting for
			// this method to return (and thus not reading these events).
			select {
			case es.Uninstall <- s:
				break uninstallLoop
			case <-s.Logs:
			case <-s.Hashes:
			case <-s.Headers:
			}
		}
	}()
}

// Error returns the error channel
func (s *Subscription) Error() <-chan error {
	return s.ErrCh
}
