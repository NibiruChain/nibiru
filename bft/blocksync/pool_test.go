package blocksync

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	cmtrand "github.com/NibiruChain/nibiru/v2/bft/libs/rand"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	"github.com/NibiruChain/nibiru/v2/bft/types"
)

func init() {
	peerTimeout = 2 * time.Second
}

type testPeer struct {
	id        p2p.ID
	base      int64
	height    int64
	inputChan chan inputData // make sure each peer's data is sequential
	malicious bool
}

type inputData struct {
	t       *testing.T
	pool    *BlockPool
	request BlockRequest
}

// Malicious nodes parameters
const MaliciousLie = 5  // This is how much the malicious node claims to be higher than the real height
const BlackholeSize = 3 // This is how many blocks the malicious node will not return (missing) above real height
const MaliciousTestMaximumLength = 5 * time.Minute

func (p testPeer) runInputRoutine() {
	go func() {
		for input := range p.inputChan {
			p.simulateInput(input)
		}
	}()
}

// Request desired, pretend like we got the block immediately.
func (p testPeer) simulateInput(input inputData) {
	block := &types.Block{Header: types.Header{Height: input.request.Height}, LastCommit: &types.Commit{}} // real blocks have LastCommit
	// If this peer is malicious
	if p.malicious {
		realHeight := p.height - MaliciousLie
		// And the requested height is above the real height
		if input.request.Height > realHeight {
			// Then provide a fake block
			block.LastCommit = nil // Fake block, no LastCommit
			// or provide no block at all, if we are close to the real height
			if input.request.Height <= realHeight+BlackholeSize {
				input.pool.RedoRequestFrom(input.request.Height, p.id)
				return
			}
		}
	}
	input.pool.AddBlock(input.request.PeerID, block, 123)
	// TODO: uncommenting this creates a race which is detected by:
	// https://github.com/golang/go/blob/2bd767b1022dd3254bcec469f0ee164024726486/src/testing/testing.go#L854-L856
	// see: https://github.com/tendermint/tendermint/issues/3390#issue-418379890
	// input.t.Logf("Added block from peer %v (height: %v)", input.request.PeerID, input.request.Height)
}

type testPeers map[p2p.ID]*testPeer

func (ps testPeers) start() {
	for _, v := range ps {
		v.runInputRoutine()
	}
}

func (ps testPeers) stop() {
	for _, v := range ps {
		close(v.inputChan)
	}
}

func makePeers(numPeers int, minHeight, maxHeight int64) testPeers {
	peers := make(testPeers, numPeers)
	for i := 0; i < numPeers; i++ {
		peerID := p2p.ID(cmtrand.Str(12))
		height := minHeight + cmtrand.Int63n(maxHeight-minHeight)
		base := minHeight + int64(i)
		if base > height {
			base = height
		}
		peers[peerID] = &testPeer{peerID, base, height, make(chan inputData, 10), false}
	}
	return peers
}

func TestBlockPoolBasic(t *testing.T) {
	var (
		start      = int64(42)
		peers      = makePeers(10, start, 1000)
		errorsCh   = make(chan peerError)
		requestsCh = make(chan BlockRequest)
	)
	pool := NewBlockPool(start, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())

	err := pool.Start()
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		if err := pool.Stop(); err != nil {
			t.Error(err)
		}
	})

	peers.start()
	defer peers.stop()

	// Introduce each peer.
	go func() {
		for _, peer := range peers {
			pool.SetPeerRange(peer.id, peer.base, peer.height)
		}
	}()

	// Start a goroutine to pull blocks
	go func() {
		for {
			if !pool.IsRunning() {
				return
			}
			first, second := pool.PeekTwoBlocks()
			if first != nil && second != nil {
				pool.PopRequest(first.Time)
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Pull from channels
	for {
		select {
		case err := <-errorsCh:
			t.Error(err)
		case request := <-requestsCh:
			t.Logf("Pulled new BlockRequest %v", request)
			if request.Height == 300 {
				return // Done!
			}

			peers[request.PeerID].inputChan <- inputData{t, pool, request}
		}
	}
}

func TestBlockPoolTimeout(t *testing.T) {
	var (
		start      = int64(42)
		peers      = makePeers(10, start, 1000)
		errorsCh   = make(chan peerError)
		requestsCh = make(chan BlockRequest)
	)

	pool := NewBlockPool(start, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())
	err := pool.Start()
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() {
		if err := pool.Stop(); err != nil {
			t.Error(err)
		}
	})

	for _, peer := range peers {
		t.Logf("Peer %v", peer.id)
	}

	// Introduce each peer.
	go func() {
		for _, peer := range peers {
			// Force uniform base so every peer contributes to maxPeerHeight at pool startup.
			pool.SetPeerRange(peer.id, start, peer.height)
		}
	}()

	// Start a goroutine to pull blocks
	go func() {
		for {
			if !pool.IsRunning() {
				return
			}
			first, second := pool.PeekTwoBlocks()
			if first != nil && second != nil {
				pool.PopRequest(first.Time)
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Pull from channels
	counter := 0
	timedOut := map[p2p.ID]struct{}{}
	for {
		select {
		case err := <-errorsCh:
			t.Log(err)
			// consider error to be always timeout here
			if _, ok := timedOut[err.peerID]; !ok {
				counter++
				if counter == len(peers) {
					return // Done!
				}
			}
		case request := <-requestsCh:
			t.Logf("Pulled new BlockRequest %+v", request)
		}
	}
}

func TestBPRequesterRedoPreservesBothPeers(t *testing.T) {
	requester := newBPRequester(nil, 1)
	requester.redo("peerA")
	requester.redo("peerB")

	// Exactly one coalesced wake-up signal.
	require.Equal(t, 1, len(requester.redoCh))

	requester.mtx.Lock()
	events := requester.redoPeers
	requester.mtx.Unlock()

	peerSet := map[p2p.ID]struct{}{}
	for _, ev := range events {
		peerSet[ev.peerID] = struct{}{}
	}
	require.Contains(t, peerSet, p2p.ID("peerA"))
	require.Contains(t, peerSet, p2p.ID("peerB"))
}

// Regression test: with the old chan p2p.ID capacity-1 design, if a redo signal
// was already pending in the channel, a concurrent redo for the second peer would
// be silently dropped. Verify that no redo event is ever lost.
func TestBPRequesterRedoNeverDropsEvent(t *testing.T) {
	requester := newBPRequester(nil, 1)

	// Two redo calls for peerA fill the old capacity-2 channel, then peerB's
	// redo would have been dropped with the previous implementation.
	requester.redo("peerA")
	requester.redo("peerA")
	requester.redo("peerB")

	// Still exactly one wake-up signal (coalesced).
	require.Equal(t, 1, len(requester.redoCh))

	requester.mtx.Lock()
	events := requester.redoPeers
	requester.mtx.Unlock()

	counts := map[p2p.ID]int{}
	for _, ev := range events {
		counts[ev.peerID]++
	}
	require.Equal(t, 2, counts["peerA"])
	require.Equal(t, 1, counts["peerB"], "peerB redo must not be dropped")
}

func TestBlockPoolRemovePeer(t *testing.T) {
	peers := make(testPeers, 10)
	for i := 0; i < 10; i++ {
		peerID := p2p.ID(fmt.Sprintf("%d", i+1))
		height := int64(i + 1)
		peers[peerID] = &testPeer{peerID, 0, height, make(chan inputData), false}
	}
	requestsCh := make(chan BlockRequest)
	errorsCh := make(chan peerError)

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())
	err := pool.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := pool.Stop(); err != nil {
			t.Error(err)
		}
	})

	// add peers
	for peerID, peer := range peers {
		pool.SetPeerRange(peerID, peer.base, peer.height)
	}
	assert.EqualValues(t, 10, pool.MaxPeerHeight())

	// remove not-existing peer
	assert.NotPanics(t, func() { pool.RemovePeer(p2p.ID("Superman")) })

	// remove peer with biggest height
	pool.RemovePeer(p2p.ID("10"))
	assert.EqualValues(t, 9, pool.MaxPeerHeight())

	// remove all peers
	for peerID := range peers {
		pool.RemovePeer(peerID)
	}

	assert.EqualValues(t, 0, pool.MaxPeerHeight())
}

func TestBlockPoolMaliciousNode(t *testing.T) {
	// Setup:
	// * each peer has blocks 1..N but the malicious peer reports 1..N+5 (block N+1,N+2,N+3 missing, N+4,N+5 fake)
	// * The malicious peer is ahead of the network but not by much, so it does not get dropped from the pool
	//   with a timeout error. (If a peer does not send blocks after 2 seconds, they are disconnected.)
	// * The network creates new blocks every second. The malicious peer will also get ahead with another fake block.
	// * The pool verifies blocks every half second. This ensures that the pool catches up with the network.
	// * When the pool encounters a fake block sent by the malicious peer and has the previous block from a good peer,
	//   it can prove that the block is fake. The malicious peer gets banned, together with the sender of the previous (valid) block.
	// Additional notes:
	// * After a minute of ban, the malicious peer is unbanned. If the pool IsCaughtUp() by then and consensus started,
	//   there is no impact. If blocksync did not catch up yet, the malicious peer can continue its lie until the next ban.
	// * The pool has an initial 3 seconds spin-up time before it starts verifying peers. (So peers have a chance to
	//   connect.) If the initial height is 7 and the block creation is 1/second, verification will start at height 10.
	// * Testing with height 7, the main functionality of banning a malicious peer is tested.
	//   Testing with height 127, a malicious peer can reconnect and the subsequent banning is also tested.
	//   This takes a couple of minutes to complete, so we don't run it.
	const InitialHeight = 7
	peers := testPeers{
		p2p.ID("good"):  &testPeer{p2p.ID("good"), 1, InitialHeight, make(chan inputData), false},
		p2p.ID("bad"):   &testPeer{p2p.ID("bad"), 1, InitialHeight + MaliciousLie, make(chan inputData), true},
		p2p.ID("good1"): &testPeer{p2p.ID("good1"), 1, InitialHeight, make(chan inputData), false},
	}
	errorsCh := make(chan peerError)
	requestsCh := make(chan BlockRequest)

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())

	err := pool.Start()
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		if err := pool.Stop(); err != nil {
			t.Error(err)
		}
	})

	peers.start()
	t.Cleanup(func() { peers.stop() })

	// Simulate blocks created on each peer regularly and update pool max height.
	go func() {
		// Introduce each peer
		for _, peer := range peers {
			pool.SetPeerRange(peer.id, peer.base, peer.height)
		}
		for {
			time.Sleep(1 * time.Second) // Speed of new block creation
			for _, peer := range peers {
				peer.height += 1                                   // Network height increases on all peers
				pool.SetPeerRange(peer.id, peer.base, peer.height) // Tell the pool that a new height is available
			}
		}
	}()

	// Start a goroutine to verify blocks
	go func() {
		for {
			time.Sleep(500 * time.Millisecond) // Speed of block verification
			if !pool.IsRunning() {
				return
			}
			first, second := pool.PeekTwoBlocks()
			if first != nil && second != nil {
				if second.LastCommit == nil {
					// Second block is fake
					pool.RemovePeerAndRedoAllPeerRequests(second.Height)
				} else {
					pool.PopRequest(first.Time)
				}
			}
		}
	}()

	testTicker := time.NewTicker(200 * time.Millisecond) // speed of test execution
	t.Cleanup(func() { testTicker.Stop() })

	bannedOnce := false // true when the malicious peer was banned at least once
	startTime := time.Now()

	// Pull from channels
	for {
		select {
		case err := <-errorsCh:
			t.Error(err)
		case request := <-requestsCh:
			// Process request
			peers[request.PeerID].inputChan <- inputData{t, pool, request}
		case <-testTicker.C:
			pool.mtx.Lock()
			banned := pool.isPeerBanned("bad")
			pool.mtx.Unlock()
			bannedOnce = bannedOnce || banned // Keep bannedOnce true, even if the malicious peer gets unbanned
			caughtUp := pool.IsCaughtUp()
			// Success: pool caught up and malicious peer was banned at least once
			if caughtUp && bannedOnce {
				t.Logf("Pool caught up, malicious peer was banned at least once, start consensus.")
				return
			}
			// Failure: the pool caught up without banning the bad peer at least once
			require.False(t, caughtUp, "Network caught up without banning the malicious peer at least once.")
			// Failure: the network could not catch up in the allotted time
			require.True(t, time.Since(startTime) < MaliciousTestMaximumLength, "Network ran too long, stopping test.")
		}
	}
}

func TestBlockPoolMaliciousNodeMaxInt64(t *testing.T) {
	// Setup:
	// * each peer has blocks 1..N but the malicious peer reports 1..max(int64) (blocks N+1... do not exist)
	// * The malicious peer then reports 1..N this time
	// * Afterwards, it can choose to disconnect or stay connected to serve blocks that it has
	// * The node ends up stuck in blocksync forever because max height is never reached (as of 63a2a6458)
	// Additional notes:
	// * When a peer is removed, we only update max height if it equals peer's
	// height. The aforementioned scenario where peer reports its height twice
	// lowering the height was not accounted for.
	const initialHeight = 7
	peers := testPeers{
		p2p.ID("good"):  &testPeer{p2p.ID("good"), 1, initialHeight, make(chan inputData), false},
		p2p.ID("bad"):   &testPeer{p2p.ID("bad"), 1, math.MaxInt64, make(chan inputData), true},
		p2p.ID("good1"): &testPeer{p2p.ID("good1"), 1, initialHeight, make(chan inputData), false},
	}
	errorsCh := make(chan peerError, 3)
	requestsCh := make(chan BlockRequest)

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())

	err := pool.Start()
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		if err := pool.Stop(); err != nil {
			t.Error(err)
		}
	})

	peers.start()
	t.Cleanup(func() { peers.stop() })

	// Simulate blocks created on each peer regularly and update pool max height.
	go func() {
		// Introduce each peer
		for _, peer := range peers {
			pool.SetPeerRange(peer.id, peer.base, peer.height)
		}

		// Report the lower height
		peers["bad"].height = initialHeight
		pool.SetPeerRange(p2p.ID("bad"), 1, initialHeight)

		ticker := time.NewTicker(1 * time.Second) // Speed of new block creation
		defer ticker.Stop()
		for {
			select {
			case <-pool.Quit():
				return
			case <-ticker.C:
				for _, peer := range peers {
					peer.height++                                      // Network height increases on all peers
					pool.SetPeerRange(peer.id, peer.base, peer.height) // Tell the pool that a new height is available
				}
			}
		}
	}()

	// Start a goroutine to verify blocks
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // Speed of new block creation
		defer ticker.Stop()
		for {
			select {
			case <-pool.Quit():
				return
			case <-ticker.C:
				first, second := pool.PeekTwoBlocks()
				if first != nil && second != nil {
					if second.LastCommit == nil {
						// Second block is fake
						pool.RemovePeerAndRedoAllPeerRequests(second.Height)
					} else {
						pool.PopRequest(first.Time)
					}
				}
			}
		}
	}()

	testTicker := time.NewTicker(200 * time.Millisecond) // speed of test execution
	t.Cleanup(func() { testTicker.Stop() })

	bannedOnce := false // true when the malicious peer was banned at least once
	startTime := time.Now()

	// Pull from channels
	for {
		select {
		case err := <-errorsCh:
			if err.peerID == "bad" { // ignore errors from the malicious peer
				t.Log(err)
			} else {
				t.Error(err)
			}
		case request := <-requestsCh:
			// Process request
			peers[request.PeerID].inputChan <- inputData{t, pool, request}
		case <-testTicker.C:
			pool.mtx.Lock()
			banned := pool.isPeerBanned("bad")
			pool.mtx.Unlock()
			bannedOnce = bannedOnce || banned // Keep bannedOnce true, even if the malicious peer gets unbanned
			caughtUp := pool.IsCaughtUp()
			// Success: pool caught up and malicious peer was banned at least once
			if caughtUp && bannedOnce {
				t.Logf("Pool caught up, malicious peer was banned at least once, start consensus.")
				return
			}
			// Failure: the pool caught up without banning the bad peer at least once
			require.False(t, caughtUp, "Network caught up without banning the malicious peer at least once.")
			// Failure: the network could not catch up in the allotted time
			require.True(t, time.Since(startTime) < MaliciousTestMaximumLength, "Network ran too long, stopping test.")
		}
	}
}

// TestBlockPoolBansPeerWithBaseGreaterThanHeight verifies that a peer whose self-reported base
// exceeds its own height (a structurally impossible state) is banned.
func TestBlockPoolBansPeerWithBaseGreaterThanHeight(t *testing.T) {
	requestsCh := make(chan BlockRequest, 10)
	errorsCh := make(chan peerError, 10)

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())

	badID := p2p.ID("bad")
	pool.SetPeerRange(badID, 500, 100)

	require.True(t, pool.IsPeerBanned(badID), "peer reporting base > height must be banned")
	require.EqualValues(t, 0, pool.MaxPeerHeight(), "banned peer must not raise maxPeerHeight")
}

func TestBlockPoolFollowsAdjacentPeerRanges(t *testing.T) {
	pool := NewBlockPool(10, nil, nil, time.Second)
	pool.startTime = time.Now()

	pool.SetPeerRange("old", 1, 10)
	pool.SetPeerRange("recent", 12, 20)
	require.EqualValues(t, 10, pool.MaxPeerHeight())
	require.True(t, pool.IsCaughtUp(), "a gap leaves no successor for block 10")

	pool.SetPeerRange("bridge", 11, 11)
	require.EqualValues(t, 20, pool.MaxPeerHeight())
	require.False(t, pool.IsCaughtUp())

	pool.RemovePeer("bridge")
	require.EqualValues(t, 10, pool.MaxPeerHeight())
	require.True(t, pool.IsCaughtUp())

	pool.SetPeerRange("successor", 11, 11)
	pool.RemovePeer("recent")
	require.EqualValues(t, 11, pool.MaxPeerHeight())
	require.False(t, pool.IsCaughtUp(), "block 10 still has a successor")

	pool.requesters[10] = newBPRequester(pool, 10)
	pool.PopRequest(time.Now())
	require.True(t, pool.IsCaughtUp(), "block 11 has no successor")
}

func TestBlockPoolWaitsForFallbackWithOnlyPrunedPeers(t *testing.T) {
	const noBlockTimeout = time.Second
	pool := NewBlockPool(10, nil, nil, noBlockTimeout)
	pool.startTime = time.Now()

	pool.SetPeerRange("pruned", 20, 30)
	pool.SetPeerRange("recent", 25, 35)
	require.Zero(t, pool.MaxPeerHeight())
	require.False(t, pool.IsCaughtUp(), "handoff must wait for the fallback")

	pool.startTime = time.Now().Add(-(peerConnWait + noBlockTimeout))
	require.True(t, pool.IsCaughtUp(), "the fallback timeout has elapsed")
}

func TestBlockPoolHandsOffAtGenesisWithHeightZeroPeer(t *testing.T) {
	pool := NewBlockPool(1, nil, nil, time.Second)
	pool.startTime = time.Now()

	pool.SetPeerRange("genesis", 0, 0)
	require.Zero(t, pool.MaxPeerHeight())
	require.True(t, pool.IsCaughtUp(), "a fresh network can hand off immediately")
}

func TestBlockPoolRecordsAppliedBlockAfterRequesterRetry(t *testing.T) {
	pool := NewBlockPool(10, nil, nil, time.Second)
	requester := newBPRequester(pool, 10)
	requester.peerID = "old"
	pool.requesters[10] = requester

	appliedTime := time.Now().Add(-time.Minute)
	block := &types.Block{Header: types.Header{Height: 10, Time: appliedTime}}
	matched, stored := requester.setBlock(block, "old")
	require.True(t, matched)
	require.True(t, stored)
	first, _ := pool.PeekTwoBlocks()
	require.Same(t, block, first)

	// A retry clears the requester's candidate while the reactor holds the block.
	require.True(t, requester.reset("old"))
	require.Nil(t, requester.getBlock())
	pool.PopRequest(first.Time)

	require.EqualValues(t, 11, pool.Height())
	require.Equal(t, appliedTime, pool.syncTracker.lastBlockHeaderTime)
	require.False(t, pool.syncTracker.lastBlockWallTime.IsZero())
}

// TestPeerOnTimeoutDoesNotDeadlockOnSendError is a regression test for
// bpPeer.onTimeout holding pool.mtx while calling sendError on an unbuffered
// channel.
func TestPeerOnTimeoutDoesNotDeadlockOnSendError(t *testing.T) {
	requestsCh := make(chan BlockRequest, 10)
	errorsCh := make(chan peerError) // unbuffered: keeps onTimeout blocked in sendError

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())
	require.NoError(t, pool.Start())
	t.Cleanup(func() { _ = pool.Stop() })

	pool.SetPeerRange("A", 1, 10)
	pool.mtx.Lock()
	peer := pool.peers["A"]
	pool.mtx.Unlock()

	go peer.onTimeout()
	time.Sleep(50 * time.Millisecond)

	heightDone := make(chan struct{})
	go func() {
		pool.Height()
		close(heightDone)
	}()

	select {
	case <-heightDone:
		<-errorsCh
	case <-time.After(500 * time.Millisecond):
		<-errorsCh
		<-heightDone
		t.Fatal("deadlock: onTimeout held pool.mtx while blocked in sendError")
	}
}

// TestAddBlockDoesNotDeadlockOnSendError is a regression test for AddBlock
// holding pool.mtx while calling sendError on an unbuffered channel.
func TestAddBlockDoesNotDeadlockOnSendError(t *testing.T) {
	requestsCh := make(chan BlockRequest, 10)
	errorsCh := make(chan peerError) // unbuffered: keeps AddBlock blocked in sendError

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())
	require.NoError(t, pool.Start())
	t.Cleanup(func() { _ = pool.Stop() })

	pool.mtx.Lock()
	req := newBPRequester(pool, 1)
	req.peerID = "A"
	pool.requesters[1] = req
	pool.mtx.Unlock()

	block := &types.Block{Header: types.Header{Height: 1}, LastCommit: &types.Commit{}}

	// "B" did not request the block; setBlock fails → sendError while holding pool.mtx.
	go pool.AddBlock("B", block, 123)
	time.Sleep(50 * time.Millisecond)

	heightDone := make(chan struct{})
	go func() {
		pool.Height()
		close(heightDone)
	}()

	select {
	case <-heightDone:
		<-errorsCh
	case <-time.After(500 * time.Millisecond):
		<-errorsCh
		<-heightDone
		t.Fatal("deadlock: AddBlock held pool.mtx while blocked in sendError")
	}
}

func TestBlockPoolHasPendingRequestFrom(t *testing.T) {
	requestsCh := make(chan BlockRequest, 10)
	errorsCh := make(chan peerError, 10)

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())

	const (
		primary   = p2p.ID("primary")
		secondary = p2p.ID("secondary")
		stranger  = p2p.ID("stranger")
	)

	// check initial state
	require.False(t, pool.HasPendingRequestFrom(primary))
	require.False(t, pool.HasPendingRequestFrom(secondary))
	require.False(t, pool.HasPendingRequestFrom(stranger))

	// Install a requester for height 1 targeting `primary`. We set the
	// fields directly so we don't have to spin up the request goroutine.
	pool.mtx.Lock()
	req1 := newBPRequester(pool, 1)
	req1.peerID = primary
	pool.requesters[1] = req1
	pool.mtx.Unlock()

	require.True(t, pool.HasPendingRequestFrom(primary), "requested peer should be reported as pending")
	require.False(t, pool.HasPendingRequestFrom(stranger), "non-requested peer must not be reported as pending")

	// A second requester at height 2 also covers the secondPeerID slot.
	pool.mtx.Lock()
	req2 := newBPRequester(pool, 2)
	req2.peerID = primary
	req2.secondPeerID = secondary
	pool.requesters[2] = req2
	pool.mtx.Unlock()

	require.True(t, pool.HasPendingRequestFrom(secondary), "secondary peer slot should count as pending")

	// Removing both requesters drops the pending state.
	pool.mtx.Lock()
	delete(pool.requesters, 1)
	delete(pool.requesters, 2)
	pool.mtx.Unlock()

	require.False(t, pool.HasPendingRequestFrom(primary))
	require.False(t, pool.HasPendingRequestFrom(secondary))
}

// TestBlockPoolTwoInflatedHeightPeersStaggered verifies that two faulty peers reporting
// inflated heights and reconnecting after timeout cannot stall blocksync indefinitely.
//
// Stall scenario (affects the old maxPeerHeight-only IsCaughtUp):
//   - Two faulty peers report Height=MaxInt64, staggered so at least one is always in the pool
//   - maxPeerHeight stays at MaxInt64 forever → IsCaughtUp() never returns true
//   - Node is permanently stuck in blocksync
//
// Defense (pure timeout-based IsCaughtUp):
//   - After syncing past the honest peer's tip, no more blocks arrive
//   - lastBlockTime stops updating → timeout fires → escape
//   - The faulty peers' inflated height claims are irrelevant
func TestBlockPoolTwoInflatedHeightPeersStaggered(t *testing.T) {
	const honestHeight = int64(50)

	errorsCh := make(chan peerError, 10)
	requestsCh := make(chan BlockRequest, 100)

	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	pool.SetLogger(log.TestingLogger())
	require.NoError(t, pool.Start())
	t.Cleanup(func() { _ = pool.Stop() })

	// Set up honest peers at the real chain tip
	pool.SetPeerRange(p2p.ID("good1"), 1, honestHeight)
	pool.SetPeerRange(p2p.ID("good2"), 1, honestHeight)

	// Two faulty peers report MaxInt64:
	//   bad1  — added early, gets assigned to requesters, times out,
	//           gets removed by removeTimedoutPeers, reconnects every 100ms
	//   bad2  — added AFTER the requester creation burst (3.5s). The first
	//           80 requesters (4 peers × 20) are created at 2ms intervals
	//           after peerConnWait (3s), so by 3.5s all requesters are
	//           already cycling through bad1 with 30s retryTimer. bad2
	//           never gets assigned, never has incrPending called, and
	//           silently keeps maxPeerHeight at MaxInt64 permanently.
	const reconInterval = 100 * time.Millisecond
	stopReconn := make(chan struct{})
	defer close(stopReconn)

	pool.SetPeerRange(p2p.ID("bad1"), 1, math.MaxInt64)
	go func() {
		for {
			select {
			case <-stopReconn:
				return
			case <-time.After(reconInterval):
				pool.SetPeerRange(p2p.ID("bad1"), 1, math.MaxInt64)
			}
		}
	}()
	// bad2: silent guardian, added after requester burst
	go func() {
		select {
		case <-stopReconn:
			return
		case <-time.After(3500 * time.Millisecond):
		}
		pool.SetPeerRange(p2p.ID("bad2"), 1, math.MaxInt64)
		for {
			select {
			case <-stopReconn:
				return
			case <-time.After(reconInterval):
				pool.SetPeerRange(p2p.ID("bad2"), 1, math.MaxInt64)
			}
		}
	}()

	// Simulate block sync up to the honest peer's tip
	for h := int64(1); h <= honestHeight; h++ {
		pool.mtx.Lock()
		r := newBPRequester(pool, h)
		pool.requesters[h] = r
		r.setBlock(
			&types.Block{Header: types.Header{Height: h, Time: time.Now()}},
			r.peerID,
		)
		pool.mtx.Unlock()
		pool.PopRequest(time.Now())
	}

	require.EqualValues(t, honestHeight+1, pool.Height(),
		"pool height advanced past honest peers")

	// Drain requestsCh: the faulty peers never respond to requests,
	// forcing requesters to wait for the 30s retryTimer.
	go func() {
		for {
			if !pool.IsRunning() {
				return
			}
			select {
			case <-requestsCh:
			case <-pool.Quit():
				return
			}
		}
	}()

	// Core assertion: IsCaughtUp() must eventually return true.
	// With the original maxPeerHeight-only check, this assertion
	// FAILS because pool.height (51) >= MaxInt64-1 is never true.
	// With the timeout-based fix, IsCaughtUp() returns true after
	// noBlockTimeout (1s in tests) elapses without any new blocks.
	//
	// This is a negative test: it fails against the original code
	// and passes against the fixed code.
	t.Logf("Waiting for IsCaughtUp() to return true (maxPeerHeight=%d, height=%d)",
		pool.MaxPeerHeight(), pool.Height())
	require.Eventually(t, func() bool {
		return pool.IsCaughtUp()
	}, 10*time.Second, 50*time.Millisecond,
		"IsCaughtUp() never returned true — node would be permanently stuck in blocksync. "+
			"This is EXPECTED for the original maxPeerHeight-only check (the stall). "+
			"With the timeout-based fix, IsCaughtUp() returns true after noBlockTimeout.")

	pool.mtx.Lock()
	numPeers := len(pool.peers)
	pool.mtx.Unlock()
	t.Logf("IsCaughtUp() became true (height=%d, maxPeerHeight=%d, peers=%d) — "+
		"node would escape blocksync despite the faulty peers",
		pool.Height(), pool.MaxPeerHeight(), numPeers)
}
