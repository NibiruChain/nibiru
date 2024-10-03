package pubsub

import (
	"log"
	"sort"
	"sync"
	"testing"
	"time"

	rpccore "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// subscribeAndPublish: Helper function used to perform concurrent subscription
// and publishing actions. It concurrently subscribes multiple clients to the
// specified topic and simultanesouly sends an empty message to the topic channel
// for each subscription.
func subscribeAndPublish(t *testing.T, eb EventBus, topic string, topicChan chan rpccore.ResultEvent) {
	var (
		wg               sync.WaitGroup
		subscribersCount = 50
		emptyMsg         = rpccore.ResultEvent{}
	)
	for i := 0; i < subscribersCount; i++ {
		wg.Add(1)
		// concurrently subscribe to the topic
		go func() {
			defer wg.Done()
			_, _, err := eb.Subscribe(topic)
			require.NoError(t, err)
		}()

		// send events to the topic
		wg.Add(1)
		go func() {
			defer wg.Done()
			topicChan <- emptyMsg
		}()
	}
	wg.Wait()
}

type SuitePubsub struct {
	suite.Suite
}

func TestSuitePubsub(t *testing.T) {
	suite.Run(t, new(SuitePubsub))
}

func (s *SuitePubsub) TestAddTopic() {
	q := NewEventBus()
	// dummy vars
	topicA := "guard"
	topicB := "cream"

	s.NoError(q.AddTopic(topicA, make(<-chan rpccore.ResultEvent)))
	s.NoError(q.AddTopic(topicB, make(<-chan rpccore.ResultEvent)))
	s.Error(q.AddTopic(topicB, make(<-chan rpccore.ResultEvent)))

	topics := q.Topics()
	sort.Strings(topics) // cream should be first
	s.Require().EqualValues([]string{topicB, topicA}, topics)
}

func (s *SuitePubsub) TestSubscribe() {
	q := NewEventBus()

	// dummy vars
	topicA := "0xfoo"
	topicB := "blockchain"

	srcA := make(chan rpccore.ResultEvent)
	err := q.AddTopic(topicA, srcA)
	s.NoError(err)

	srcB := make(chan rpccore.ResultEvent)
	err = q.AddTopic(topicB, srcB)
	s.NoError(err)

	// subscriber channels
	subChanA, _, err := q.Subscribe(topicA)
	s.NoError(err)
	subChanB1, _, err := q.Subscribe(topicB)
	s.NoError(err)
	subChanB2, _, err := q.Subscribe(topicB)
	s.NoError(err)

	wg := new(sync.WaitGroup)
	wg.Add(4)

	emptyMsg := rpccore.ResultEvent{}
	go func() {
		defer wg.Done()
		msg := <-subChanA
		log.Println(topicA+":", msg)
		s.EqualValues(emptyMsg, msg)
	}()

	go func() {
		defer wg.Done()
		msg := <-subChanB1
		log.Println(topicB+":", msg)
		s.EqualValues(emptyMsg, msg)
	}()

	go func() {
		defer wg.Done()
		msg := <-subChanB2
		log.Println(topicB+"2:", msg)
		s.EqualValues(emptyMsg, msg)
	}()

	go func() {
		defer wg.Done()

		time.Sleep(time.Second)

		close(srcA)
		close(srcB)
	}()

	wg.Wait()
	time.Sleep(time.Second)
}

// TestConcurrentSubscribeAndPublish: Stress tests the module to make sure that
// operations are handled properly under concurrent access.
func (s *SuitePubsub) TestConcurrentSubscribeAndPublish() {
	var (
		wg        sync.WaitGroup
		eb        = NewEventBus()
		topicName = "topic-name"
		topicCh   = make(chan rpccore.ResultEvent)
		runsCount = 5
	)

	err := eb.AddTopic(topicName, topicCh)
	s.Require().NoError(err)

	for i := 0; i < runsCount; i++ {
		subscribeAndPublish(s.T(), eb, topicName, topicCh)
	}

	// close channel to make test end
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		close(topicCh)
	}()

	wg.Wait()
}
