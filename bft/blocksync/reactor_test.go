package blocksync

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cfg "github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	mpmocks "github.com/NibiruChain/nibiru/v2/bft/mempool/mocks"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	p2pmocks "github.com/NibiruChain/nibiru/v2/bft/p2p/mocks"
	bcproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/blocksync"
	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	"github.com/NibiruChain/nibiru/v2/bft/proxy"
	sm "github.com/NibiruChain/nibiru/v2/bft/state"
	"github.com/NibiruChain/nibiru/v2/bft/store"
	"github.com/NibiruChain/nibiru/v2/bft/types"
	cmttime "github.com/NibiruChain/nibiru/v2/bft/types/time"
)

var config *cfg.Config

func randGenesisDoc(numValidators int, randPower bool, minPower int64) (*types.GenesisDoc, []types.PrivValidator) {
	validators := make([]types.GenesisValidator, numValidators)
	privValidators := make([]types.PrivValidator, numValidators)
	for i := 0; i < numValidators; i++ {
		val, privVal := types.RandValidator(randPower, minPower)
		validators[i] = types.GenesisValidator{
			PubKey: val.PubKey,
			Power:  val.VotingPower,
		}
		privValidators[i] = privVal
	}
	sort.Sort(types.PrivValidatorsByAddress(privValidators))

	return &types.GenesisDoc{
		GenesisTime: cmttime.Now(),
		ChainID:     config.ChainID(),
		Validators:  validators,
	}, privValidators
}

func genesisDocWithValsPowers(powers []int64) (*types.GenesisDoc, []types.PrivValidator) {
	if len(powers) == 0 {
		panic("must have atleast 1 validator")
	}

	validators := make([]types.GenesisValidator, len(powers))
	privValidators := make([]types.PrivValidator, len(powers))
	for i, power := range powers {
		val, privVal := types.RandValidator(false, power)
		validators[i] = types.GenesisValidator{
			PubKey: val.PubKey,
			Power:  val.VotingPower,
		}
		privValidators[i] = privVal
	}
	sort.Sort(types.PrivValidatorsByAddress(privValidators))

	return &types.GenesisDoc{
		GenesisTime: cmttime.Now(),
		ChainID:     config.ChainID(),
		Validators:  validators,
	}, privValidators
}

type ReactorPair struct {
	reactor *Reactor
	app     proxy.AppConns
}

func newReactor(
	t *testing.T,
	logger log.Logger,
	genDoc *types.GenesisDoc,
	privVals []types.PrivValidator,
	maxBlockHeight int64,
	opts ...reactorOption,
) ReactorPair {
	var options reactorOpts
	for _, opt := range opts {
		opt(&options)
	}

	app := &testApp{}
	cc := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(cc, proxy.NopMetrics())
	err := proxyApp.Start()
	if err != nil {
		panic(fmt.Errorf("error start app: %w", err))
	}

	blockDB := dbm.NewMemDB()
	stateDB := dbm.NewMemDB()
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	blockStore := store.NewBlockStore(blockDB)

	state, err := stateStore.LoadFromDBOrGenesisDoc(genDoc)
	if err != nil {
		panic(fmt.Errorf("error constructing state from genesis file: %w", err))
	}

	mp := &mpmocks.Mempool{}
	mp.On("Lock").Return()
	mp.On("Unlock").Return()
	mp.On("FlushAppConn", mock.Anything).Return(nil)
	mp.On("Update",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(nil)

	// Make the Reactor itself.
	// NOTE we have to create and commit the blocks first because
	// pool.height is determined from the store.
	blockSync := true
	db := dbm.NewMemDB()
	stateStore = sm.NewStore(db, sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	blockExec := sm.NewBlockExecutor(stateStore, log.TestingLogger(), proxyApp.Consensus(),
		mp, sm.EmptyEvidencePool{})
	if err = stateStore.Save(state); err != nil {
		panic(err)
	}

	// let's add some blocks in
	for blockHeight := int64(1); blockHeight <= maxBlockHeight; blockHeight++ {
		lastCommit := types.NewCommit(blockHeight-1, 0, types.BlockID{}, nil)
		if blockHeight > 1 {
			lastBlockMeta := blockStore.LoadBlockMeta(blockHeight - 1)
			lastBlock := blockStore.LoadBlock(blockHeight - 1)

			// Simulate commits for the previous height
			commitSigs := make([]types.CommitSig, len(privVals))
			for _, val := range privVals {
				pubKey, err := val.GetPubKey()
				if err != nil {
					panic(err)
				}
				idx, _ := state.Validators.GetByAddress(pubKey.Address())

				vote, err := types.MakeVote(
					lastBlock.Header.Height,
					lastBlockMeta.BlockID,
					state.Validators,
					val,
					lastBlock.Header.ChainID,
					time.Now(),
				)
				if err != nil {
					panic(err)
				}
				commitSigs[idx] = vote.CommitSig()
			}
			if options.invalidLastCommitSigBlock == blockHeight {
				// Corrupt the signature of the last validator in the set. With
				// the right powers, the commit still carries 2/3+ of valid power.
				commitSigs[len(commitSigs)-1].Signature = []byte("invalid signature")
			}
			lastCommit = types.NewCommit(lastBlock.Header.Height, 0,
				lastBlockMeta.BlockID, commitSigs)
		}

		thisBlock, err := state.MakeBlock(blockHeight, nil, lastCommit, nil, state.Validators.Proposer.Address)
		require.NoError(t, err)

		thisParts, err := thisBlock.MakePartSet(types.BlockPartSizeBytes)
		require.NoError(t, err)
		blockID := types.BlockID{Hash: thisBlock.Hash(), PartSetHeader: thisParts.Header()}

		if options.invalidLastCommitSigBlock == blockHeight {
			// ApplyBlock would reject the corrupted LastCommit.
			state, _, err = blockExec.ApplyVerifiedBlock(state, blockID, thisBlock)
		} else {
			state, _, err = blockExec.ApplyBlock(state, blockID, thisBlock)
		}
		if err != nil {
			panic(fmt.Errorf("error apply block: %w", err))
		}

		blockStore.SaveBlock(thisBlock, thisParts, lastCommit)
	}

	bcReactor := NewReactor(state.Copy(), blockExec, blockStore, blockSync)
	bcReactor.SetLogger(logger.With("module", "blockchain"))

	return ReactorPair{bcReactor, proxyApp}
}

type reactorOpts struct {
	invalidLastCommitSigBlock int64
}

type reactorOption func(*reactorOpts)

// withInvalidLastCommitSigBlock corrupts one signature in the LastCommit of
// the block at the given height.
func withInvalidLastCommitSigBlock(height int64) reactorOption {
	return func(o *reactorOpts) {
		o.invalidLastCommitSigBlock = height
	}
}

func TestNoBlockResponse(t *testing.T) {
	config = cfg.ResetTestRoot("blockchain_reactor_test")
	defer os.RemoveAll(config.RootDir)
	genDoc, privVals := randGenesisDoc(1, false, 30)

	maxBlockHeight := int64(65)

	reactorPairs := make([]ReactorPair, 2)

	reactorPairs[0] = newReactor(t, log.TestingLogger(), genDoc, privVals, maxBlockHeight)
	reactorPairs[1] = newReactor(t, log.TestingLogger(), genDoc, privVals, 0)

	p2p.MakeConnectedSwitches(config.P2P, 2, func(i int, s *p2p.Switch) *p2p.Switch {
		s.AddReactor("BLOCKCHAIN", reactorPairs[i].reactor)
		return s
	}, p2p.Connect2Switches)

	defer func() {
		for _, r := range reactorPairs {
			err := r.reactor.Stop()
			require.NoError(t, err)
			err = r.app.Stop()
			require.NoError(t, err)
		}
	}()

	tests := []struct {
		height   int64
		existent bool
	}{
		{maxBlockHeight + 2, false},
		{10, true},
		{1, true},
		{100, false},
	}

	for {
		if reactorPairs[1].reactor.pool.IsCaughtUp() {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, maxBlockHeight, reactorPairs[0].reactor.store.Height())

	for _, tt := range tests {
		block := reactorPairs[1].reactor.store.LoadBlock(tt.height)
		if tt.existent {
			assert.True(t, block != nil)
		} else {
			assert.True(t, block == nil)
		}
	}
}

// NOTE: This is too hard to test without
// an easy way to add test peer to switch
// or without significant refactoring of the module.
// Alternatively we could actually dial a TCP conn but
// that seems extreme.
func TestBadBlockStopsPeer(t *testing.T) {
	config = cfg.ResetTestRoot("blockchain_reactor_test")
	defer os.RemoveAll(config.RootDir)
	genDoc, privVals := randGenesisDoc(1, false, 30)

	maxBlockHeight := int64(148)

	// Other chain needs a different validator set
	otherGenDoc, otherPrivVals := randGenesisDoc(1, false, 30)
	otherChain := newReactor(t, log.TestingLogger(), otherGenDoc, otherPrivVals, maxBlockHeight)

	defer func() {
		err := otherChain.reactor.Stop()
		require.Error(t, err)
		err = otherChain.app.Stop()
		require.NoError(t, err)
	}()

	reactorPairs := make([]ReactorPair, 4)

	reactorPairs[0] = newReactor(t, log.TestingLogger(), genDoc, privVals, maxBlockHeight)
	reactorPairs[1] = newReactor(t, log.TestingLogger(), genDoc, privVals, 0)
	reactorPairs[2] = newReactor(t, log.TestingLogger(), genDoc, privVals, 0)
	reactorPairs[3] = newReactor(t, log.TestingLogger(), genDoc, privVals, 0)

	switches := p2p.MakeConnectedSwitches(config.P2P, 4, func(i int, s *p2p.Switch) *p2p.Switch {
		s.AddReactor("BLOCKCHAIN", reactorPairs[i].reactor)
		return s
	}, p2p.Connect2Switches)

	defer func() {
		for _, r := range reactorPairs {
			err := r.reactor.Stop()
			require.NoError(t, err)

			err = r.app.Stop()
			require.NoError(t, err)
		}
	}()

	for {
		time.Sleep(1 * time.Second)
		caughtUp := true
		for _, r := range reactorPairs {
			if !r.reactor.pool.IsCaughtUp() {
				caughtUp = false
			}
		}
		if caughtUp {
			break
		}
	}

	// at this time, reactors[0-3] is the newest
	assert.Equal(t, 3, reactorPairs[1].reactor.Switch.Peers().Size())

	// Mark reactorPairs[3] as an invalid peer. Fiddling with .store without a mutex is a data
	// race, but can't be easily avoided.
	reactorPairs[3].reactor.store = otherChain.reactor.store

	lastReactorPair := newReactor(t, log.TestingLogger(), genDoc, privVals, 0)
	reactorPairs = append(reactorPairs, lastReactorPair)

	switches = append(switches, p2p.MakeConnectedSwitches(config.P2P, 1, func(i int, s *p2p.Switch) *p2p.Switch {
		s.AddReactor("BLOCKCHAIN", reactorPairs[len(reactorPairs)-1].reactor)
		return s
	}, p2p.Connect2Switches)...)

	for i := 0; i < len(reactorPairs)-1; i++ {
		p2p.Connect2Switches(switches, i, len(reactorPairs)-1)
	}

	for {
		if lastReactorPair.reactor.pool.IsCaughtUp() || lastReactorPair.reactor.Switch.Peers().Size() == 0 {
			break
		}

		time.Sleep(1 * time.Second)
	}

	assert.True(t, lastReactorPair.reactor.Switch.Peers().Size() < len(reactorPairs)-1)
}

type testApp struct {
	abci.BaseApplication
}

var _ abci.Application = (*testApp)(nil)

func (app *testApp) Info(req abci.RequestInfo) (resInfo abci.ResponseInfo) {
	return abci.ResponseInfo{}
}

func (app *testApp) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return abci.ResponseBeginBlock{}
}

func (app *testApp) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	return abci.ResponseEndBlock{}
}

func (app *testApp) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	return abci.ResponseDeliverTx{Events: []abci.Event{}}
}

func (app *testApp) CheckTx(req abci.RequestCheckTx) abci.ResponseCheckTx {
	return abci.ResponseCheckTx{}
}

func (app *testApp) Commit() abci.ResponseCommit {
	return abci.ResponseCommit{}
}

func (app *testApp) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	return
}

// TestInvalidSignatureAfterTwoThirdsIsRejected checks that blocksync fully
// verifies second.LastCommit before it stores that commit as the seen commit
// for first. A light check would stop counting once 2/3+ of the power signed
// and would miss the invalid signature that follows.
func TestInvalidSignatureAfterTwoThirdsIsRejected(t *testing.T) {
	config = cfg.ResetTestRoot("blockchain_reactor_test")
	defer os.RemoveAll(config.RootDir)
	genDoc, privVals := genesisDocWithValsPowers([]int64{10, 10, 10, 1})

	const (
		maxBlockHeight = int64(10)
		invalidHeight  = int64(5)
	)

	reactorPairs := make([]ReactorPair, 2)
	reactorPairs[0] = newReactor(t, log.TestingLogger(), genDoc, privVals, maxBlockHeight,
		withInvalidLastCommitSigBlock(invalidHeight))
	reactorPairs[1] = newReactor(t, log.TestingLogger(), genDoc, privVals, 0)

	p2p.MakeConnectedSwitches(config.P2P, 2, func(i int, s *p2p.Switch) *p2p.Switch {
		s.AddReactor("BLOCKCHAIN", reactorPairs[i].reactor)
		return s
	}, p2p.Connect2Switches)

	defer func() {
		for _, r := range reactorPairs {
			_ = r.reactor.Stop()
			_ = r.app.Stop()
		}
	}()

	syncing := reactorPairs[1].reactor
	require.Eventually(t, func() bool {
		return syncing.Switch.Peers().Size() == 0
	}, 30*time.Second, 50*time.Millisecond, "the peer serving the bad commit must be stopped")

	// Block invalidHeight-1 must not be stored, since its seen commit would be
	// the corrupted LastCommit of block invalidHeight.
	require.Equal(t, invalidHeight-2, syncing.store.Height())

	vals := reactorPairs[0].reactor.initialState.Validators
	for h := int64(1); h <= syncing.store.Height(); h++ {
		seenCommit := syncing.store.LoadSeenCommit(h)
		require.NotNil(t, seenCommit)
		require.NoError(t, vals.VerifyCommit(genDoc.ChainID, seenCommit.BlockID, h, seenCommit),
			"seen commit at height %d", h)
	}
}

// TestSwitchToBlockSyncRaceWithSetPeerRange detects the data race between
// Reactor.SwitchToBlockSync() writing pool.height without pool.mtx and
// SetPeerRange() reading it via updateMaxPeerHeight() under pool.mtx.
// Must be run with -race to be effective.
func TestSwitchToBlockSyncRaceWithSetPeerRange(t *testing.T) {
	requestsCh := make(chan BlockRequest, 1000)
	errorsCh := make(chan peerError, 1000)
	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	require.NoError(t, pool.Start())
	t.Cleanup(func() { _ = pool.Stop() })

	r := &Reactor{pool: pool}
	r.Logger = log.TestingLogger()

	state := sm.State{LastBlockHeight: 100}

	start := make(chan struct{})
	var wg sync.WaitGroup

	// SwitchToBlockSync() writes pool.height; pool.Start returns
	// ErrAlreadyStarted since the pool is pre-started, but the write still
	// races with concurrent reads.
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		_ = r.SwitchToBlockSync(state)
	}()

	// SetPeerRange holds pool.mtx and reads pool.height via updateMaxPeerHeight.
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := int64(0); i < 500; i++ {
			pool.SetPeerRange("peer1", i+2, i+100)
		}
	}()

	close(start)
	wg.Wait()
}

func TestSwitchToBlockSyncRecomputesMaxPeerHeight(t *testing.T) {
	requestsCh := make(chan BlockRequest, 1000)
	errorsCh := make(chan peerError, 1000)
	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	require.NoError(t, pool.Start())
	t.Cleanup(func() { _ = pool.Stop() })

	// base=2 > pool.height=1: excluded from maxPeerHeight, so it stays 0.
	pool.SetPeerRange("peer1", 2, 200)
	require.Equal(t, int64(0), pool.MaxPeerHeight())

	r := &Reactor{pool: pool}
	r.Logger = log.TestingLogger()

	_ = r.SwitchToBlockSync(sm.State{LastBlockHeight: 100})
	require.Equal(t, int64(200), pool.MaxPeerHeight())
}

// newFilterReactor builds a minimal Reactor wired to a started BlockPool,
// suitable for exercising FilterMsgBytes without spinning up the full p2p
// stack.
func newFilterReactor(t *testing.T) *Reactor {
	t.Helper()

	requestsCh := make(chan BlockRequest, 1000)
	errorsCh := make(chan peerError, 1000)
	pool := NewBlockPool(1, requestsCh, errorsCh, 1*time.Second)
	require.NoError(t, pool.Start())
	t.Cleanup(func() { _ = pool.Stop() })

	r := &Reactor{pool: pool}
	r.blockSync.Store(true)
	return r
}

// seedRequester inserts a bpRequester targeting peerID at the given height,
// bypassing makeRequestersRoutine so the test can drive pool state directly.
func seedRequester(r *Reactor, height int64, peerID p2p.ID) {
	req := newBPRequester(r.pool, height)
	req.peerID = peerID
	r.pool.mtx.Lock()
	r.pool.requesters[height] = req
	r.pool.mtx.Unlock()
}

func mockPeer(id p2p.ID) *p2pmocks.Peer {
	p := &p2pmocks.Peer{}
	p.On("ID").Return(id).Maybe()
	return p
}

func mockBlockRequestPeer(id p2p.ID, queued bool) *p2pmocks.Peer {
	peer := mockPeer(id)
	peer.On("String").Return(string(id)).Maybe()
	var requests any
	peer.On("Set", sentBlockRequestsKey, mock.Anything).Run(func(args mock.Arguments) {
		requests = args.Get(1)
	})
	peer.On("Get", sentBlockRequestsKey).Return(func(string) any { return requests })
	peer.On("TrySendEnvelope", mock.Anything).Return(queued)
	return peer
}

func TestFilterAllowsOnlyRequestedLateReplyDuringSync(t *testing.T) {
	config = cfg.ResetTestRoot("blocksync_late_reply")
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(config.RootDir)) })
	genDoc, privVals := genesisDocWithValsPowers([]int64{30})
	source := newReactor(t, log.TestingLogger(), genDoc, privVals, 3)
	t.Cleanup(func() { require.NoError(t, source.app.Stop()) })
	block := source.reactor.store.LoadBlock(1)
	require.NotNil(t, block)
	wireBlock, err := block.ToProto()
	require.NoError(t, err)
	reply := &bcproto.BlockResponse{Block: wireBlock}
	wireReply, err := proto.Marshal(reply.Wrap())
	require.NoError(t, err)

	r := newFilterReactor(t)
	r.Logger = log.TestingLogger()
	delayed := r.InitPeer(mockBlockRequestPeer("delayed", true))
	failed := r.InitPeer(mockBlockRequestPeer("failed", false))
	other := r.InitPeer(mockBlockRequestPeer("other", true))
	require.True(t, sentRequestsFor(delayed).send(delayed, block.Height))
	require.False(t, sentRequestsFor(failed).send(failed, block.Height))
	require.True(t, sentRequestsFor(other).send(other, block.Height+1))

	// The first peer's copy was applied while the second peer's reply was in flight.
	r.ReceiveEnvelope(p2p.Envelope{
		ChannelID: BlocksyncChannel, Src: delayed,
		Message: &bcproto.StatusResponse{Base: 1, Height: 3},
	})
	r.pool.mtx.Lock()
	r.pool.peers[delayed.ID()].incrPending()
	r.pool.mtx.Unlock()
	seedRequester(r, block.Height, "first")
	r.pool.PopRequest(block.Time)
	seedRequester(r, block.Height+1, other.ID())
	require.True(t, r.pool.IsRunning())
	expiry := sentRequestsFor(delayed).expiresAt[block.Height]
	require.NoError(t, r.filterMsgBytesAt(BlocksyncChannel, delayed, wireReply, expiry.Add(-time.Nanosecond)))
	require.ErrorContains(t, r.FilterMsgBytes(BlocksyncChannel, failed, wireReply), "unsolicited")
	require.ErrorContains(t, r.FilterMsgBytes(BlocksyncChannel, other, wireReply), "unsolicited")
	reconnected := r.InitPeer(mockBlockRequestPeer(delayed.ID(), true))
	require.ErrorContains(t, r.FilterMsgBytes(BlocksyncChannel, reconnected, wireReply), "unsolicited")

	envelope := p2p.Envelope{ChannelID: BlocksyncChannel, Src: delayed, Message: reply}
	r.receiveEnvelopeAt(envelope, expiry)
	r.pool.mtx.Lock()
	pending := r.pool.peers[delayed.ID()].numPending
	r.pool.mtx.Unlock()
	require.Zero(t, pending, "the delayed reply satisfies the peer's request")
	require.ErrorContains(t, r.FilterMsgBytes(BlocksyncChannel, delayed, wireReply), "unsolicited")

	// A second delivery must not release the slot for a different request.
	require.True(t, sentRequestsFor(delayed).send(delayed, block.Height+2))
	r.pool.mtx.Lock()
	r.pool.peers[delayed.ID()].incrPending()
	r.pool.mtx.Unlock()
	r.receiveEnvelopeAt(envelope, expiry)
	r.pool.mtx.Lock()
	pending = r.pool.peers[delayed.ID()].numPending
	r.pool.mtx.Unlock()
	require.Equal(t, int32(1), pending)
	r.RemovePeer(delayed, nil)
	r.RemovePeer(other, nil)
	require.Nil(t, sentRequestsFor(other))
}

func TestLateBlockReplyDoesNotSettleReplacementConnection(t *testing.T) {
	r := newFilterReactor(t)
	r.Logger = log.TestingLogger()
	const peerID p2p.ID = "reconnecting"
	oldPeer := r.InitPeer(mockBlockRequestPeer(peerID, true))
	newPeer := r.InitPeer(mockBlockRequestPeer(peerID, true))
	addPeerRange := func(peer p2p.Peer) {
		r.ReceiveEnvelope(p2p.Envelope{
			ChannelID: BlocksyncChannel, Src: peer,
			Message: &bcproto.StatusResponse{Base: 1, Height: 3},
		})
	}
	addPeerRange(oldPeer)
	require.NotNil(t, r.pool.pickIncrAvailablePeer(1, ""))
	oldRequests := sentRequestsFor(oldPeer)
	require.True(t, oldRequests.send(oldPeer, 1))

	block := &types.Block{Header: types.Header{Height: 1, Time: time.Now()}}
	seedRequester(r, block.Height, "first")
	r.pool.PopRequest(block.Time)
	require.True(t, oldRequests.admit(block.Height, time.Now()))
	require.Equal(t, block.Height, oldRequests.takeAdmission())

	// The connection closes after receive has taken the admission, while
	// delivery is still in progress.
	r.RemovePeer(oldPeer, nil)
	addPeerRange(newPeer)
	replacement := r.pool.pickIncrAvailablePeer(2, "")
	require.NotNil(t, replacement)
	t.Cleanup(func() { r.RemovePeer(newPeer, nil) })
	require.True(t, sentRequestsFor(newPeer).send(newPeer, 2))

	r.pool.addBlock(peerID, block, 123, oldRequests, true)
	r.pool.mtx.Lock()
	pending := replacement.numPending
	r.pool.mtx.Unlock()
	require.EqualValues(t, 1, pending, "the replacement's request is still pending")
}

func TestActiveBlockReplySettlesOnlyItsConnection(t *testing.T) {
	r := newFilterReactor(t)
	r.Logger = log.TestingLogger()
	const peerID p2p.ID = "reconnecting"
	oldPeer := r.InitPeer(mockBlockRequestPeer(peerID, true))
	newPeer := r.InitPeer(mockBlockRequestPeer(peerID, true))
	register := func(peer p2p.Peer) {
		r.ReceiveEnvelope(p2p.Envelope{
			ChannelID: BlocksyncChannel, Src: peer,
			Message: &bcproto.StatusResponse{Base: 1, Height: 3},
		})
	}
	register(oldPeer)
	oldRequests := sentRequestsFor(oldPeer)
	require.NotNil(t, r.pool.pickIncrAvailablePeer(1, ""))
	require.True(t, oldRequests.send(oldPeer, 1))
	require.True(t, oldRequests.admit(1, time.Now()))
	require.EqualValues(t, 1, oldRequests.takeAdmission())

	r.RemovePeer(oldPeer, nil)
	register(newPeer)
	t.Cleanup(func() { r.RemovePeer(newPeer, nil) })
	replacement := r.pool.pickIncrAvailablePeer(1, "")
	require.NotNil(t, replacement)
	require.Same(t, replacement, r.pool.pickIncrAvailablePeer(2, ""))
	newRequests := sentRequestsFor(newPeer)
	require.True(t, newRequests.send(newPeer, 1))
	require.True(t, newRequests.send(newPeer, 2))
	seedRequester(r, 1, peerID)
	r.pool.numPending = 1

	assertPending := func(peerSlots, blocks int32) {
		t.Helper()
		r.pool.mtx.Lock()
		peerPending := replacement.numPending
		r.pool.mtx.Unlock()
		_, blockPending, _ := r.pool.GetStatus()
		require.Equal(t, peerSlots, peerPending)
		require.Equal(t, blocks, blockPending)
	}
	block := &types.Block{Header: types.Header{Height: 1}}
	r.pool.addBlock(peerID, block, 123, oldRequests, true)
	assertPending(2, 1)
	first, _ := r.pool.PeekTwoBlocks()
	require.Nil(t, first)

	require.True(t, newRequests.take(1, time.Now()))
	r.pool.addBlock(peerID, block, 123, newRequests, true)
	assertPending(1, 0)
	first, _ = r.pool.PeekTwoBlocks()
	require.Same(t, block, first)

	requested := newRequests.take(1, time.Now())
	require.False(t, requested)
	r.pool.addBlock(peerID, block, 123, newRequests, requested)
	assertPending(1, 0)
}

func TestSentBlockRequestExpiresAtRetryInterval(t *testing.T) {
	var requests sentBlockRequests
	now := time.Now()
	requests.remember(1, now)
	requests.remember(2, now)
	require.True(t, requests.admit(1, now.Add(sentBlockRequestTTL-time.Nanosecond)))
	require.False(t, requests.admit(2, now.Add(sentBlockRequestTTL)))
	require.EqualValues(t, 1, requests.takeAdmission())
	require.Zero(t, requests.takeAdmission())
	require.Empty(t, requests.expiresAt)
}

func TestFilterMsgBytes(t *testing.T) {
	wireBytesFor := func(t *testing.T, m *bcproto.Message) []byte {
		t.Helper()
		b, err := proto.Marshal(m)
		require.NoError(t, err)
		require.NotEmpty(t, b)
		return b
	}

	blockResponseBytes := func(t *testing.T) []byte {
		return wireBytesFor(t, &bcproto.Message{
			Sum: &bcproto.Message_BlockResponse{
				BlockResponse: &bcproto.BlockResponse{Block: &cmtproto.Block{}},
			},
		})
	}

	blockRequestBytes := func(t *testing.T) []byte {
		return wireBytesFor(t, &bcproto.Message{
			Sum: &bcproto.Message_BlockRequest{
				BlockRequest: &bcproto.BlockRequest{Height: 1},
			},
		})
	}

	const expected p2p.ID = "expected"
	const unexpected p2p.ID = "unexpected"

	tests := []struct {
		name      string
		setup     func(t *testing.T) *Reactor // returns a configured reactor
		chID      byte
		peer      p2p.ID
		bytesFn   func(t *testing.T) []byte
		expectErr string // substring; "" means no error
	}{
		{
			name: "rejects BlockResponse when blocksync never ran",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				r.blockSync.Store(false)
				return r
			},
			chID:      BlocksyncChannel,
			peer:      unexpected,
			bytesFn:   blockResponseBytes,
			expectErr: "blocksync not active",
		},
		{
			// After catching up, the pool stops but in-flight responses from our
			// own requests arrive after the switch to consensus. The peer is
			// honest and must not be disconnected.
			name: "allows late BlockResponse after pool stops for consensus switch",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				require.NoError(t, r.pool.Stop())
				return r
			},
			chID:    BlocksyncChannel,
			peer:    unexpected,
			bytesFn: blockResponseBytes,
		},
		{
			// Any connected peer can reach this path once the pool is stopped,
			// so the sig-count guard must still apply here.
			name: "rejects oversized BlockResponse after pool stops for consensus switch",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				require.NoError(t, r.pool.Stop())
				return r
			},
			chID:      BlocksyncChannel,
			peer:      unexpected,
			bytesFn:   func(t *testing.T) []byte { return blockResponseBytesWithSigs(t, types.MaxVotesCount+1) },
			expectErr: "too many commit signatures",
		},
		{
			name:      "rejects unsolicited BlockResponse with no requesters",
			setup:     newFilterReactor,
			chID:      BlocksyncChannel,
			peer:      unexpected,
			bytesFn:   blockResponseBytes,
			expectErr: "unsolicited BlockResponse from peer unexpected",
		},
		{
			name: "rejects BlockResponse from peer we did not request from",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				seedRequester(r, 1, expected)
				return r
			},
			chID:      BlocksyncChannel,
			peer:      unexpected,
			bytesFn:   blockResponseBytes,
			expectErr: "unsolicited BlockResponse from peer unexpected",
		},
		{
			name: "allows BlockResponse from solicited peer",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				seedRequester(r, 1, expected)
				return r
			},
			chID:    BlocksyncChannel,
			peer:    expected,
			bytesFn: blockResponseBytes,
		},
		{
			name:    "allows non-BlockResponse messages even when disabled",
			setup:   newFilterReactor,
			chID:    BlocksyncChannel,
			peer:    "any",
			bytesFn: blockRequestBytes,
		},
		{
			name:    "ignores other channels",
			setup:   newFilterReactor,
			chID:    byte(0x20),
			peer:    "any",
			bytesFn: blockResponseBytes,
		},
		{
			name:    "ignores empty bytes",
			setup:   newFilterReactor,
			chID:    BlocksyncChannel,
			peer:    "any",
			bytesFn: func(*testing.T) []byte { return nil },
		},
		{
			name: "allows BlockResponse at MaxVotesCount commit signatures",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				seedRequester(r, 1, expected)
				return r
			},
			chID:    BlocksyncChannel,
			peer:    expected,
			bytesFn: func(t *testing.T) []byte { return blockResponseBytesWithSigs(t, types.MaxVotesCount) },
		},
		{
			name: "rejects BlockResponse exceeding commit signature cap",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				seedRequester(r, 1, expected)
				return r
			},
			chID:      BlocksyncChannel,
			peer:      expected,
			bytesFn:   func(t *testing.T) []byte { return blockResponseBytesWithSigs(t, types.MaxVotesCount+1) },
			expectErr: "too many commit signatures",
		},
		{
			name: "rejects BlockResponse splitting signatures across duplicate Block fields",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				seedRequester(r, 1, expected)
				return r
			},
			chID: BlocksyncChannel,
			peer: expected,
			bytesFn: func(t *testing.T) []byte {
				half := types.MaxVotesCount/2 + 1 // 2*half > MaxVotesCount
				a := blockResponseBytesWithSigs(t, half)
				b := blockResponseBytesWithSigs(t, half)
				return append(append([]byte{}, a...), b...)
			},
			expectErr: "too many commit signatures",
		},
		{
			name: "rejects BlockResponse when first byte is not BlockResponse proto tag",
			setup: func(t *testing.T) *Reactor {
				r := newFilterReactor(t)
				seedRequester(r, 1, expected)
				return r
			},
			chID: BlocksyncChannel,
			peer: expected,
			bytesFn: func(t *testing.T) []byte {
				// Prepend an empty BlockRequest field (tag 0x0a, len 0)
				// so msgBytes[0] != BlockResponse oneof tag, then append
				// a real BlockResponse payload that exceeds the cap.
				oversized := blockResponseBytesWithSigs(t, types.MaxVotesCount+1)
				return append([]byte{0x0a, 0x00}, oversized...)
			},
			expectErr: "too many commit signatures",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.setup(t)
			peer := mockPeer(tc.peer)
			peer.On("Get", sentBlockRequestsKey).Return(nil).Maybe()
			err := r.FilterMsgBytes(tc.chID, peer, tc.bytesFn(t))
			if tc.expectErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectErr)
		})
	}
}

func TestStubUnmarshalAllocs(t *testing.T) {
	tests := []struct {
		name       string
		numCommits int
	}{
		{"10k commit sigs", 10_000},
		{"100k commit sigs", 100_000},
		{"1m commit sigs", 1_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := blockResponseBytesWithSigs(t, tt.numCommits)
			allocs := testing.AllocsPerRun(20, func() {
				var stub bcproto.SigCountMessage
				require.NoError(t, stub.Unmarshal(payload))
				require.Len(t, stub.BlockResponse.Block.LastCommit.Signatures, tt.numCommits)
			})
			const maxAllocs = 50
			require.LessOrEqualf(t, int(allocs), maxAllocs, "unmarshal allocated %d times, more than max allowed", int(allocs), maxAllocs)
		})
	}
}

func blockResponseBytesWithSigs(t *testing.T, commitSigs int) []byte {
	t.Helper()
	commit := &cmtproto.Commit{Signatures: make([]cmtproto.CommitSig, commitSigs)}
	for i := range commit.Signatures {
		commit.Signatures[i] = cmtproto.CommitSig{BlockIdFlag: cmtproto.BlockIDFlagAbsent}
	}

	msg := &bcproto.Message{
		Sum: &bcproto.Message_BlockResponse{
			BlockResponse: &bcproto.BlockResponse{
				Block: &cmtproto.Block{LastCommit: commit},
			},
		},
	}

	payload, err := proto.Marshal(msg)
	require.NoError(t, err)
	return payload
}

func TestPeerNotDisconnectedOnLateBlockResponseAfterConsensusSwitch(t *testing.T) {
	config = cfg.ResetTestRoot("blockchain_reactor_test")
	defer os.RemoveAll(config.RootDir)

	const nBlocks = int64(10)
	genDoc, privVals := genesisDocWithValsPowers([]int64{30})

	servingPair := newReactor(t, log.TestingLogger(), genDoc, privVals, nBlocks)
	defer func() {
		require.NoError(t, servingPair.reactor.Stop())
		require.NoError(t, servingPair.app.Stop())
	}()

	syncingPair := newReactor(t, log.TestingLogger(), genDoc, privVals, 0)
	defer func() {
		require.NoError(t, syncingPair.reactor.Stop())
		require.NoError(t, syncingPair.app.Stop())
	}()

	switches := p2p.MakeConnectedSwitches(config.P2P, 2, func(i int, s *p2p.Switch) *p2p.Switch {
		if i == 0 {
			s.AddReactor("BLOCKCHAIN", syncingPair.reactor)
		} else {
			s.AddReactor("BLOCKCHAIN", servingPair.reactor)
		}
		return s
	}, p2p.Connect2Switches)

	require.Eventually(t, func() bool {
		return !syncingPair.reactor.pool.IsRunning()
	}, 30*time.Second, 5*time.Millisecond, "syncing pool did not stop")

	// Pool stopped (consensus switch). Simulate a late in-flight response by
	// sending block 1 from the serving node — exercises the onReceive →
	// FilterMsgBytes path in p2p/peer.go.
	block := servingPair.reactor.store.LoadBlock(1)
	require.NotNil(t, block)
	bl, err := block.ToProto()
	require.NoError(t, err)

	peers := switches[1].Peers().List()
	require.Len(t, peers, 1)
	require.True(t, peers[0].TrySendEnvelope(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message:   &bcproto.BlockResponse{Block: bl},
	}), "message must be queued to exercise the filter path")

	require.Never(t, func() bool {
		return switches[0].Peers().Size() != 1
	}, 200*time.Millisecond, 5*time.Millisecond,
		"serving peer was incorrectly disconnected by a late in-flight BlockResponse")
}

// TestSwitchToBlockSyncDoesNotEscapeBeforeFirstBlock checks that after state
// sync the sync tracker does not end blocksync before any block is synced.
// SwitchToBlockSync moves pool.height past the height the reactor was built
// with, so startHeight must move with it.
func TestSwitchToBlockSyncDoesNotEscapeBeforeFirstBlock(t *testing.T) {
	requestsCh := make(chan BlockRequest, 1000)
	errorsCh := make(chan peerError, 1000)
	pool := NewBlockPool(1, requestsCh, errorsCh, defaultNoBlockTimeout)
	pool.SetLogger(log.TestingLogger())

	r := &Reactor{pool: pool}
	r.Logger = log.TestingLogger()

	require.NoError(t, pool.Start())
	t.Cleanup(func() { _ = pool.Stop() })

	// pool.Start returns ErrAlreadyStarted, so no pool routine is spawned, but
	// pool.height and pool.startHeight are still set.
	_ = r.SwitchToBlockSync(sm.State{LastBlockHeight: 100})

	pool.SetPeerRange("peer1", 1, 200)

	// Let peerConnWait pass without syncing a block. The peer never answers.
	time.Sleep(peerConnWait + 200*time.Millisecond)

	require.False(t, pool.IsCaughtUp(),
		"blocksync must not end before the first block after state sync")
}
