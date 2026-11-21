package blocksync

import (
	"time"

	cmtsync "github.com/NibiruChain/nibiru/v2/bft/libs/sync"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	bcproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/blocksync"
)

const (
	sentBlockRequestsKey = "BlocksyncSentBlockRequests"
	sentBlockRequestTTL  = requestRetrySeconds * time.Second
)

// sentBlockRequests belongs to one peer connection. It holds successfully
// queued requests awaiting a reply, even after the pool retires a requester.
type sentBlockRequests struct {
	mtx            cmtsync.Mutex
	expiresAt      map[int64]time.Time
	admittedHeight int64 // The peer receive loop delivers one message at a time.
}

func sentRequestsFor(peer p2p.Peer) *sentBlockRequests {
	requests, _ := peer.Get(sentBlockRequestsKey).(*sentBlockRequests)
	return requests
}

func (requests *sentBlockRequests) send(peer p2p.Peer, height int64) bool {
	if requests == nil {
		return false
	}
	requests.mtx.Lock()
	defer requests.mtx.Unlock()

	// Hold the same lock as the receive filter so a fast reply can see the
	// result of the queue operation. TrySendEnvelope is nonblocking.
	if !peer.TrySendEnvelope(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message:   &bcproto.BlockRequest{Height: height},
	}) {
		return false
	}

	requests.remember(height, time.Now())
	return true
}

// remember keeps at most one entry per height and bounds the connection's
// history by the pool's per-peer request limit. The caller holds mtx.
func (requests *sentBlockRequests) remember(height int64, now time.Time) {
	if requests.expiresAt == nil {
		requests.expiresAt = make(map[int64]time.Time)
	}
	delete(requests.expiresAt, height)
	var oldestHeight int64
	var oldestExpiry time.Time
	for sentHeight, expiry := range requests.expiresAt {
		if !now.Before(expiry) {
			delete(requests.expiresAt, sentHeight)
		} else if oldestExpiry.IsZero() || expiry.Before(oldestExpiry) {
			oldestHeight, oldestExpiry = sentHeight, expiry
		}
	}
	limit := maxPendingRequestsPerPeer
	if requests.admittedHeight != 0 {
		limit--
	}
	if len(requests.expiresAt) >= limit {
		delete(requests.expiresAt, oldestHeight)
	}
	requests.expiresAt[height] = now.Add(sentBlockRequestTTL)
}

// admit moves a timely request out of the expiring history until delivery.
func (requests *sentBlockRequests) admit(height int64, now time.Time) bool {
	if requests == nil {
		return false
	}
	requests.mtx.Lock()
	defer requests.mtx.Unlock()
	expiry := requests.expiresAt[height]
	delete(requests.expiresAt, height)
	if !now.Before(expiry) {
		return false
	}
	requests.admittedHeight = height
	return true
}

func (requests *sentBlockRequests) takeAdmission() int64 {
	if requests == nil {
		return 0
	}
	requests.mtx.Lock()
	defer requests.mtx.Unlock()
	height := requests.admittedHeight
	requests.admittedHeight = 0
	return height
}

func (requests *sentBlockRequests) take(height int64, now time.Time) bool {
	if requests == nil {
		return false
	}
	requests.mtx.Lock()
	defer requests.mtx.Unlock()
	expiry := requests.expiresAt[height]
	delete(requests.expiresAt, height)
	return now.Before(expiry)
}

func (requests *sentBlockRequests) clear() {
	if requests == nil {
		return
	}
	requests.mtx.Lock()
	defer requests.mtx.Unlock()
	// A reply admitted before the handoff can still be in delivery.
	requests.expiresAt = nil
}
