package blocksync

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/NibiruChain/nibiru/v2/bft/crypto"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	bcproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/blocksync"
	sm "github.com/NibiruChain/nibiru/v2/bft/state"
	"github.com/NibiruChain/nibiru/v2/bft/store"
	"github.com/NibiruChain/nibiru/v2/bft/types"
)

const (
	// BlocksyncChannel is a channel for blocks and status updates (`BlockStore` height)
	BlocksyncChannel = byte(0x40)

	trySyncIntervalMS = 10

	// stop syncing when last block's time is
	// within this much of the system time.
	// stopSyncingDurationMinutes = 10

	// ask for best height every 10s
	statusUpdateIntervalSeconds = 10
	// check if we should switch to consensus reactor
	switchToConsensusIntervalSeconds = 1

	// defaultNoBlockTimeout is how long blocksync waits without syncing a
	// block before it switches to consensus.
	defaultNoBlockTimeout = 30 * time.Second
)

type consensusReactor interface {
	// for when we switch from blockchain reactor and block sync to
	// the consensus machine
	SwitchToConsensus(state sm.State, skipWAL bool)
}

type peerError struct {
	err        error
	peerID     p2p.ID
	connection *sentBlockRequests
}

func (e peerError) Error() string {
	return fmt.Sprintf("error with peer %v: %s", e.peerID, e.err.Error())
}

// Reactor implements MsgBytesFilter
var _ p2p.MsgBytesFilter = (*Reactor)(nil)

// Reactor handles long-term catchup syncing.
type Reactor struct {
	p2p.BaseReactor

	// immutable
	initialState sm.State

	blockExec *sm.BlockExecutor
	store     *store.BlockStore
	pool      *BlockPool
	blockSync atomic.Bool // enabled at start or via SwitchToBlockSync
	localAddr crypto.Address

	requestsCh <-chan BlockRequest
	errorsCh   <-chan peerError
}

func NewReactorWithOfflineStateSyncAndAddr(state sm.State, blockExec *sm.BlockExecutor, store *store.BlockStore,
	blockSync bool, localAddr crypto.Address, offlineStateSyncHeight int64) *Reactor {

	storeHeight := store.Height()
	if storeHeight == 0 {
		// If state sync was performed offline and the stores were bootstrapped to height H
		// the state store's lastHeight will be H while blockstore's Height and Base are still 0
		// 1. This scenario should not lead to a panic in this case, which is indicated by
		// having a OfflineStateSyncHeight > 0
		// 2. We need to instruct the blocksync reactor to start fetching blocks from H+1
		// instead of 0.
		storeHeight = offlineStateSyncHeight
	}
	if state.LastBlockHeight != storeHeight {
		panic(fmt.Sprintf("state (%v) and store (%v) height mismatch, stores were left in an inconsistent state", state.LastBlockHeight,
			storeHeight))
	}

	// It's okay to block since sendRequest is called from a separate goroutine
	// (bpRequester#requestRoutine; 1 per each peer).
	requestsCh := make(chan BlockRequest)

	const capacity = 1000                      // must be bigger than peers count
	errorsCh := make(chan peerError, capacity) // so we don't block in #Receive#pool.AddBlock

	startHeight := storeHeight + 1
	if startHeight == 1 {
		startHeight = state.InitialHeight
	}
	pool := NewBlockPool(startHeight, requestsCh, errorsCh, defaultNoBlockTimeout)

	bcR := &Reactor{
		initialState: state,
		blockExec:    blockExec,
		store:        store,
		pool:         pool,
		localAddr:    localAddr,
		requestsCh:   requestsCh,
		errorsCh:     errorsCh,
	}
	bcR.blockSync.Store(blockSync)
	bcR.BaseReactor = *p2p.NewBaseReactor("Reactor", bcR)
	return bcR
}

func NewReactorWithOfflineStateSync(state sm.State, blockExec *sm.BlockExecutor, store *store.BlockStore,
	blockSync bool, offlineStateSyncHeight int64) *Reactor {
	return NewReactorWithOfflineStateSyncAndAddr(state, blockExec, store, blockSync, nil, offlineStateSyncHeight)
}

// NewReactor returns new reactor instance.
func NewReactor(state sm.State, blockExec *sm.BlockExecutor, store *store.BlockStore,
	blockSync bool) *Reactor {

	return NewReactorWithOfflineStateSync(state, blockExec, store, blockSync, 0)

}

// SetLogger implements service.Service by setting the logger on reactor and pool.
func (bcR *Reactor) SetLogger(l log.Logger) {
	bcR.BaseService.Logger = l
	bcR.pool.Logger = l
}

// OnStart implements service.Service.
func (bcR *Reactor) OnStart() error {
	if bcR.blockSync.Load() {
		err := bcR.pool.Start()
		if err != nil {
			return err
		}
		go bcR.poolRoutine(false)
	}
	return nil
}

// SwitchToBlockSync is called by the state sync reactor when switching to block sync.
func (bcR *Reactor) SwitchToBlockSync(state sm.State) error {
	bcR.blockSync.Store(true)
	bcR.initialState = state

	bcR.pool.mtx.Lock()
	bcR.pool.height = state.LastBlockHeight + 1
	// The pool starts at the state sync height, not at the height the reactor
	// was built with. Otherwise IsCaughtUp sees pool.height > startHeight
	// before any block is synced and lets the sync tracker end blocksync.
	bcR.pool.startHeight = bcR.pool.height
	bcR.pool.updateMaxPeerHeight()
	bcR.pool.mtx.Unlock()

	err := bcR.pool.Start()
	if err != nil {
		return err
	}
	go bcR.poolRoutine(true)
	return nil
}

// OnStop implements service.Service.
func (bcR *Reactor) OnStop() {
	if bcR.blockSync.Load() {
		if err := bcR.pool.Stop(); err != nil {
			bcR.Logger.Error("Error stopping pool", "err", err)
		}
	}
}

// GetChannels implements Reactor
func (bcR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  BlocksyncChannel,
			Priority:            5,
			SendQueueCapacity:   1000,
			RecvBufferCapacity:  50 * 4096,
			RecvMessageCapacity: MaxMsgSize,
			MessageType:         &bcproto.Message{},
		},
	}
}

// InitPeer initializes the request history for this connection.
func (bcR *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	peer.Set(sentBlockRequestsKey, &sentBlockRequests{})
	return peer
}

// AddPeer implements Reactor by sending our state to peer.
func (bcR *Reactor) AddPeer(peer p2p.Peer) {
	peer.SendEnvelope(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message: &bcproto.StatusResponse{
			Base:   bcR.store.Base(),
			Height: bcR.store.Height(),
		},
	})
	// it's OK if send fails. will try later in poolRoutine

	// peer is added to the pool once we receive the first
	// bcStatusResponseMessage from the peer and call pool.SetPeerRange
}

// RemovePeer implements Reactor by removing peer from the pool.
func (bcR *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	requests := sentRequestsFor(peer)
	peer.Set(sentBlockRequestsKey, nil)
	if requests != nil {
		bcR.pool.removePeerConnection(peer.ID(), requests)
	}
}

// respondToPeer loads a block and sends it to the requesting peer,
// if we have it. Otherwise, we'll respond saying we don't have it.
func (bcR *Reactor) respondToPeer(msg *bcproto.BlockRequest,
	src p2p.Peer) (queued bool) {

	block := bcR.store.LoadBlock(msg.Height)
	if block != nil {
		bl, err := block.ToProto()
		if err != nil {
			bcR.Logger.Error("could not convert msg to protobuf", "err", err)
			return false
		}

		return src.TrySendEnvelope(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.BlockResponse{Block: bl},
		})
	}

	bcR.Logger.Info("Peer asking for a block we don't have", "src", src, "height", msg.Height)
	return src.TrySendEnvelope(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message:   &bcproto.NoBlockResponse{Height: msg.Height},
	})
}

// FilterMsgBytes implements p2p.MsgBytesFilter and rejects messages from
// unexpected peers before unmarshalling the request.
func (bcR *Reactor) FilterMsgBytes(chID byte, src p2p.Peer, msgBytes []byte) error {
	return bcR.filterMsgBytesAt(chID, src, msgBytes, time.Now())
}

func (bcR *Reactor) filterMsgBytesAt(chID byte, src p2p.Peer, msgBytes []byte, now time.Time) error {
	// do not check invalid messages, will fail unmarshalling
	if chID != BlocksyncChannel || len(msgBytes) == 0 {
		return nil
	}

	// unmarshal into custom stub struct that will do no allocations so we can
	// quickly and cheaply check the validity of BlockResponse message
	var stub bcproto.SigCountMessage
	if err := stub.Unmarshal(msgBytes); err != nil {
		return fmt.Errorf("malformed blocksync message from peer %s: %w", src.ID(), err)
	}
	if stub.BlockResponse == nil {
		// Not a BlockResponse oneof case, no extra validation to do in this
		// case
		return nil
	}

	// Never ran blocksync on this node — any BlockResponse is unsolicited.
	if !bcR.blockSync.Load() {
		return errors.New("unsolicited BlockResponse: blocksync not active")
	}
	// Pool has stopped (switched to consensus). Requests we sent before the
	// transition are still in flight; the peers are honest and must not be
	// disconnected for answering our own requests. Still enforce the sig-count
	// guard below, since any connected peer can reach this path now.
	if !bcR.pool.IsRunning() {
		return validateMaxVotes(stub.BlockResponse)
	}

	height := stub.BlockResponse.GetBlock().GetHeader().GetHeight()
	retired, pending := bcR.pool.responseRequestStatus(src.ID(), height)
	if !retired && !pending {
		return fmt.Errorf("unsolicited BlockResponse from peer %s", src.ID())
	}

	// validate the commit count in the response
	if err := validateMaxVotes(stub.BlockResponse); err != nil {
		return fmt.Errorf("validating max votes in BlockResponse from peer %s: %w", src.ID(), err)
	}

	// A requester can retire before delivery, even if it was live here.
	admitted := sentRequestsFor(src).admit(height, now)
	if retired && !admitted {
		return fmt.Errorf("unsolicited BlockResponse from peer %s", src.ID())
	}

	return nil
}

// validateMaxVotes validates that the number of commit signatures is less
// than the MaxVotesCount, returns an error if not.
func validateMaxVotes(br *bcproto.SigCountBlockResponse) error {
	commitSigs := 0
	if br != nil && br.Block != nil && br.Block.LastCommit != nil {
		commitSigs = len(br.Block.LastCommit.Signatures)
	}

	if commitSigs > types.MaxVotesCount {
		return fmt.Errorf("too many commit signatures: %d (max %d)", commitSigs, types.MaxVotesCount)
	}

	return nil
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (bcR *Reactor) ReceiveEnvelope(e p2p.Envelope) {
	bcR.receiveEnvelopeAt(e, time.Now())
}

func (bcR *Reactor) receiveEnvelopeAt(e p2p.Envelope, now time.Time) {
	requests := sentRequestsFor(e.Src)
	admittedHeight := requests.takeAdmission()
	if err := ValidateMsg(e.Message); err != nil {
		bcR.Logger.Error("Peer sent us invalid msg", "peer", e.Src, "msg", e.Message, "err", err)
		bcR.Switch.StopPeerForError(e.Src, err)
		return
	}

	bcR.Logger.Debug("Receive", "e.Src", e.Src, "chID", e.ChannelID, "msg", e.Message)

	switch msg := e.Message.(type) {
	case *bcproto.BlockRequest:
		bcR.respondToPeer(msg, e.Src)
	case *bcproto.BlockResponse:
		bi, err := types.BlockFromProto(msg.Block)
		if err != nil {
			bcR.Logger.Error("Block content is invalid", "err", err)
			return
		}
		if requests == nil {
			return
		}
		requested := admittedHeight == bi.Height || requests.take(bi.Height, now)
		bcR.pool.addBlock(e.Src.ID(), bi, msg.Block.Size(), requests, requested)
	case *bcproto.StatusRequest:
		// Send peer our state.
		e.Src.TrySendEnvelope(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message: &bcproto.StatusResponse{
				Height: bcR.store.Height(),
				Base:   bcR.store.Base(),
			},
		})
	case *bcproto.StatusResponse:
		// Got a peer status. Unverified.
		bcR.pool.setPeerRange(e.Src.ID(), msg.Base, msg.Height, e.Src)
	case *bcproto.NoBlockResponse:
		requests.take(msg.Height, now)
		bcR.Logger.Debug("Peer does not have requested block", "peer", e.Src, "height", msg.Height)
		if requests != nil {
			bcR.pool.redoRequestFrom(msg.Height, e.Src.ID(), requests)
		}
	default:
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

func (bcR *Reactor) localNodeBlocksTheChain(state sm.State) bool {
	_, val := state.Validators.GetByAddress(bcR.localAddr)
	if val == nil {
		return false
	}
	total := state.Validators.TotalVotingPower()
	return val.VotingPower >= total/3
}

// Handle messages from the poolReactor telling the reactor what to do.
// NOTE: Don't sleep in the FOR_LOOP or otherwise slow it down!
func (bcR *Reactor) poolRoutine(stateSynced bool) {

	trySyncTicker := time.NewTicker(trySyncIntervalMS * time.Millisecond)
	defer trySyncTicker.Stop()

	statusUpdateTicker := time.NewTicker(statusUpdateIntervalSeconds * time.Second)
	defer statusUpdateTicker.Stop()

	switchToConsensusTicker := time.NewTicker(switchToConsensusIntervalSeconds * time.Second)
	defer switchToConsensusTicker.Stop()

	blocksSynced := uint64(0)

	chainID := bcR.initialState.ChainID
	state := bcR.initialState

	lastHundred := time.Now()
	lastRate := 0.0

	didProcessCh := make(chan struct{}, 1)

	go func() {
		defer func() {
			for _, peer := range bcR.Switch.Peers().List() {
				sentRequestsFor(peer).clear()
			}
		}()
		for {
			select {
			case <-bcR.Quit():
				return
			case <-bcR.pool.Quit():
				return
			case request := <-bcR.requestsCh:
				peer := bcR.Switch.Peers().Get(request.PeerID)
				if peer == nil {
					continue
				}
				requests := sentRequestsFor(peer)
				if requests == nil || requests != request.connection {
					continue
				}
				queued := requests.send(peer, request.Height)
				if !queued {
					bcR.Logger.Debug("Send queue is full, drop block request", "peer", peer.ID(), "height", request.Height)
				}
			case err := <-bcR.errorsCh:
				peer := bcR.Switch.Peers().Get(err.peerID)
				if peer != nil && err.connection == sentRequestsFor(peer) {
					bcR.Switch.StopPeerForError(peer, err)
				}

			case <-statusUpdateTicker.C:
				// ask for status updates
				go bcR.BroadcastStatusRequest()

			}
		}
	}()

FOR_LOOP:
	for {
		select {
		case <-switchToConsensusTicker.C:
			height, numPending, lenRequesters := bcR.pool.GetStatus()
			outbound, inbound, _ := bcR.Switch.NumPeers()
			bcR.Logger.Debug("Consensus ticker", "numPending", numPending, "total", lenRequesters,
				"outbound", outbound, "inbound", inbound)
			if bcR.pool.IsCaughtUp() || bcR.localNodeBlocksTheChain(state) {
				bcR.Logger.Info("Time to switch to consensus reactor!", "height", height)
				if err := bcR.pool.Stop(); err != nil {
					bcR.Logger.Error("Error stopping pool", "err", err)
				}
				conR, ok := bcR.Switch.Reactor("CONSENSUS").(consensusReactor)
				if ok {
					conR.SwitchToConsensus(state, blocksSynced > 0 || stateSynced)
				}
				// else {
				// should only happen during testing
				// }

				break FOR_LOOP
			}

		case <-trySyncTicker.C: // chan time
			select {
			case didProcessCh <- struct{}{}:
			default:
			}

		case <-didProcessCh:
			// NOTE: It is a subtle mistake to process more than a single block
			// at a time (e.g. 10) here, because we only TrySend 1 request per
			// loop.  The ratio mismatch can result in starving of blocks, a
			// sudden burst of requests and responses, and repeat.
			// Consequently, it is better to split these routines rather than
			// coupling them as it's written here.  TODO uncouple from request
			// routine.

			// See if there are any blocks to sync.
			first, second := bcR.pool.PeekTwoBlocks()
			// bcR.Logger.Info("TrySync peeked", "first", first, "second", second)
			if first == nil || second == nil {
				// We need both to sync the first block.
				continue FOR_LOOP
			} else {
				// Try again quickly next loop.
				didProcessCh <- struct{}{}
			}

			firstParts, err := first.MakePartSet(types.BlockPartSizeBytes)
			if err != nil {
				bcR.Logger.Error("failed to make ",
					"height", first.Height,
					"err", err.Error())
				break FOR_LOOP
			}
			firstPartSetHeader := firstParts.Header()
			firstID := types.BlockID{Hash: first.Hash(), PartSetHeader: firstPartSetHeader}
			// Fully verify second.LastCommit to ensure all signatures are valid.
			// It is persisted below as the seen commit for first.
			err = state.Validators.VerifyCommit(chainID, firstID, first.Height, second.LastCommit)
			if err != nil {
				bcR.handleValidationFailure(first, second, err)
				continue FOR_LOOP
			}

			// Verify the current block against state before persisting it.
			if err = bcR.blockExec.ValidateBlock(state, first); err != nil {
				bcR.handleValidationFailure(first, second, err)
				continue FOR_LOOP
			}

			// TODO: batch saves so we dont persist to disk every block
			bcR.store.SaveBlock(first, firstParts, second.LastCommit)

			// TODO: same thing for app - but we would need a way to
			// get the hash without persisting the state
			state, _, err = bcR.blockExec.ApplyVerifiedBlock(state, firstID, first)
			if err != nil {
				// TODO This is bad, are we zombie?
				panic(fmt.Sprintf("Failed to process committed block (%d:%X): %v", first.Height, first.Hash(), err))
			}
			bcR.pool.PopRequest(first.Time)
			blocksSynced++

			if blocksSynced%100 == 0 {
				lastRate = 0.9*lastRate + 0.1*(100/time.Since(lastHundred).Seconds())
				bcR.Logger.Info("Block Sync Rate", "height", bcR.pool.height,
					"max_peer_height", bcR.pool.MaxPeerHeight(), "blocks/s", lastRate)
				lastHundred = time.Now()
			}

			continue FOR_LOOP

		case <-bcR.Quit():
			break FOR_LOOP
		}
	}
}

func (bcR *Reactor) handleValidationFailure(blockA, blockB *types.Block, err error) {
	bcR.Logger.Error("Error in validation", "height", blockA.Height, "hash", blockA.Hash(), "err", err)

	err = fmt.Errorf("Reactor validation error: %v", err)

	idA := bcR.pool.RemovePeerAndRedoAllPeerRequests(blockA.Height)
	if peerA := bcR.Switch.Peers().Get(idA); peerA != nil {
		// NOTE: we've already removed the peer's request, but we
		// still need to clean up the rest.
		bcR.Switch.StopPeerForError(peerA, err)
	}

	idB := bcR.pool.RemovePeerAndRedoAllPeerRequests(blockB.Height)
	if idA == idB {
		return
	}

	if peerB := bcR.Switch.Peers().Get(idB); peerB != nil {
		// NOTE: we've already removed the peer's request, but we
		// still need to clean up the rest.
		bcR.Switch.StopPeerForError(peerB, err)
	}
}

// BroadcastStatusRequest broadcasts `BlockStore` base and height.
func (bcR *Reactor) BroadcastStatusRequest() {
	bcR.Switch.BroadcastEnvelope(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message:   &bcproto.StatusRequest{},
	})
}
