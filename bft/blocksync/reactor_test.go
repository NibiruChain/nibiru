package blocksync

import (
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cfg "github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	mpmocks "github.com/NibiruChain/nibiru/v2/bft/mempool/mocks"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
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
