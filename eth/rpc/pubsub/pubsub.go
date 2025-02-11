// Copyright (c) 2023-2024 Nibi, Inc.
package pubsub

import (
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

type UnsubscribeFunc func()

// EventBus manages topics and subscriptions. A "topic" is a named channel of
// communication. A "subscription" is the action taken by a subscriber to express
// interest in receiving messages broadcasted from a specific topic.
type EventBus interface {
	// AddTopic: Adds a new topic with the specified name and message source
	AddTopic(name string, src <-chan coretypes.ResultEvent) error
	// RemoveTopic: Removes the specified topic and all its related data,
	// ensuring clean up of resources.
	RemoveTopic(name string)
	Subscribe(name string) (<-chan coretypes.ResultEvent, UnsubscribeFunc, error)
	Topics() []string
}

// memEventBus is an implemention of the `EventBus` interface.
type memEventBus struct {
	topics          map[string]<-chan coretypes.ResultEvent
	topicsMux       *sync.RWMutex
	subscribers     map[string]map[uint64]chan<- coretypes.ResultEvent
	subscribersMux  *sync.RWMutex
	currentUniqueID uint64
}

// NewEventBus returns a fresh imlpemention of `memEventBus`, which implements
// the `EventBus` interface for managing Ethereum topics and subscriptions.
func NewEventBus() EventBus {
	return &memEventBus{
		topics:         make(map[string]<-chan coretypes.ResultEvent),
		topicsMux:      new(sync.RWMutex),
		subscribers:    make(map[string]map[uint64]chan<- coretypes.ResultEvent),
		subscribersMux: new(sync.RWMutex),
	}
}

// GenUniqueID atomically increments and returns a unique identifier for a new subscriber.
// This ID is used internally to manage subscriber-specific channels.
func (m *memEventBus) GenUniqueID() uint64 {
	return atomic.AddUint64(&m.currentUniqueID, 1)
}

// Topics returns a list of all topics currently managed by the EventBus. The
// list is safe for concurrent access and is a snapshot of current topic names.
func (m *memEventBus) Topics() (topics []string) {
	m.topicsMux.RLock()
	defer m.topicsMux.RUnlock()

	topics = make([]string, 0, len(m.topics))
	for topicName := range m.topics {
		topics = append(topics, topicName)
	}

	return topics
}

// AddTopic adds a new topic with the specified name and message source
func (m *memEventBus) AddTopic(name string, src <-chan coretypes.ResultEvent) error {
	m.topicsMux.RLock()
	_, ok := m.topics[name]
	m.topicsMux.RUnlock()

	if ok {
		return errors.New("topic already registered")
	}

	m.topicsMux.Lock()
	m.topics[name] = src
	m.topicsMux.Unlock()

	go m.publishTopic(name, src)

	return nil
}

// RemoveTopic: Removes the specified topic and all its related data, ensuring
// clean up of resources.
func (m *memEventBus) RemoveTopic(name string) {
	m.topicsMux.Lock()
	delete(m.topics, name)
	m.topicsMux.Unlock()
}

// Subscribe attempts to create a subscription to the specified topic. It returns
// a channel to receive messages, a function to unsubscribe, and an error if the
// topic does not exist.
func (m *memEventBus) Subscribe(name string) (<-chan coretypes.ResultEvent, UnsubscribeFunc, error) {
	m.topicsMux.RLock()
	_, ok := m.topics[name]
	m.topicsMux.RUnlock()

	if !ok {
		return nil, nil, errors.Errorf("topic not found: %s", name)
	}

	ch := make(chan coretypes.ResultEvent)
	m.subscribersMux.Lock()
	defer m.subscribersMux.Unlock()

	id := m.GenUniqueID()
	if _, ok := m.subscribers[name]; !ok {
		m.subscribers[name] = make(map[uint64]chan<- coretypes.ResultEvent)
	}
	m.subscribers[name][id] = ch

	unsubscribe := func() {
		m.subscribersMux.Lock()
		defer m.subscribersMux.Unlock()
		delete(m.subscribers[name], id)
	}

	return ch, unsubscribe, nil
}

func (m *memEventBus) publishTopic(name string, src <-chan coretypes.ResultEvent) {
	for {
		msg, ok := <-src
		if !ok {
			m.closeAllSubscribers(name)
			m.topicsMux.Lock()
			delete(m.topics, name)
			m.topicsMux.Unlock()
			return
		}
		m.publishAllSubscribers(name, msg)
	}
}

// closeAllSubscribers closes all subscriber channels associated with the
// specified topic and removes the topic from the subscribers map. This function
// is typically called when a topic is deleted or no longer available to ensure
// all resources are released properly and to prevent goroutine leaks. It ensures
// thread-safe execution by locking around the operation.
func (m *memEventBus) closeAllSubscribers(name string) {
	m.subscribersMux.Lock()
	defer m.subscribersMux.Unlock()

	subscribers := m.subscribers[name]
	delete(m.subscribers, name)
	// #nosec G705
	for _, sub := range subscribers {
		close(sub)
	}
}

// publishAllSubscribers sends a message to all subscribers of the specified
// topic. It uses a non-blocking send operation to deliver the message to
// subscriber channels. If a subscriber's channel is not ready to receive the
// message (i.e., the channel is full), the message is skipped for that
// subscriber to avoid blocking the publisher. This function ensures thread-safe
// access to subscribers by using a read lock.
func (m *memEventBus) publishAllSubscribers(name string, msg coretypes.ResultEvent) {
	m.subscribersMux.RLock()
	defer m.subscribersMux.RUnlock()
	subscribers := m.subscribers[name]
	// #nosec G705
	for _, sub := range subscribers {
		select {
		case sub <- msg:
		default:
		}
	}
}
