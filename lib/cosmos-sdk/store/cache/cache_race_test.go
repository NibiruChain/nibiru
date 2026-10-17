package cache_test

// Regression tests for the inter-block cache race that halted the
// nibiru-testnet-2-fullnode-2 node at height 8658454 with
// "CONSENSUS FAILURE!!! ... wrong Block.Header.LastResultsHash".
//
// CommitKVStoreCache.Get is a read-through: on a cache miss it reads the
// underlying CommitKVStore and then caches the result (including nil misses)
// with no synchronization against concurrent Set/Delete. gRPC Service/Simulate
// traffic executes against checkState, whose CacheMultiStore branches over the
// same shared CommitKVStoreCache instances that block commit writes through.
// A simulation read racing a commit can therefore overwrite a freshly
// committed value with a stale nil; every subsequent block execution on that
// node then reads the poisoned entry while the real value sits intact in IAVL
// on disk, and the node diverges from consensus.
//
// Upstream report of the same race: https://github.com/cosmos/cosmos-sdk/issues/23891

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cosmos/iavl"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/cache"
	iavlstore "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/iavl"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
)

// gatedCommitKVStore wraps a CommitKVStore so a test can suspend a reader in
// the window between its read of the underlying store and
// CommitKVStoreCache.Get caching that result. This is the goroutine
// preemption window in which the incident's Simulate read was suspended while
// the next block committed.
type gatedCommitKVStore struct {
	types.CommitKVStore
	armed       atomic.Bool
	readStarted chan struct{}
	readResume  chan struct{}
}

func (s *gatedCommitKVStore) Get(key []byte) []byte {
	value := s.CommitKVStore.Get(key)
	if s.armed.CompareAndSwap(true, false) {
		s.readStarted <- struct{}{}
		<-s.readResume
	}
	return value
}

func newGatedCacheStore(t *testing.T) (*gatedCommitKVStore, types.CommitKVStore, *cache.CommitKVStoreCache) {
	t.Helper()
	db := dbm.NewMemDB()
	tree, err := iavl.NewMutableTree(db, 100, false)
	require.NoError(t, err)
	underlying := iavlstore.UnsafeNewStore(tree)
	gated := &gatedCommitKVStore{
		CommitKVStore: underlying,
		readStarted:   make(chan struct{}),
		readResume:    make(chan struct{}),
	}
	return gated, underlying, cache.NewCommitKVStoreCache(gated, cache.DefaultCommitKVStoreCacheSize)
}

// TestCommitKVStoreCacheStaleNilPoisoning reproduces the exact interleaving
// behind the testnet-2 fullnode-2 consensus failure at height 8658454:
//
//  1. A Simulate read misses the cache and reads "not found" from the
//     underlying store (trade 6329's initial_acc_fees, not yet committed).
//  2. The block commit writes the key through the cache to disk.
//  3. The suspended read resumes and caches its stale nil over the fresh value.
//  4. The next block's execution reads nil from the cache although the value
//     is on disk, fails a tx that every other node executed successfully, and
//     the node halts on LastResultsHash divergence one block later.
//
// This test fails on the current unsynchronized cache and must pass once
// cache reads are synchronized against write-through commits.
func TestCommitKVStoreCacheStaleNilPoisoning(t *testing.T) {
	gated, underlying, ckv := newGatedCacheStore(t)

	key := []byte("initial_acc_fees/trade_6329")
	value := []byte("committed_at_8658453")

	// (1) The simulation read misses the cache, reads nil from the
	// underlying store, and is suspended before caching the result.
	gated.armed.Store(true)
	simulateDone := make(chan []byte, 1)
	go func() { simulateDone <- ckv.Get(key) }()
	<-gated.readStarted

	// (2) The block commit writes the key through the cache while the
	// simulation read is suspended. On the unsynchronized cache the Set
	// completes inside the suspension window; a synchronized cache may
	// legitimately block it until the read finishes, so don't wait forever.
	setDone := make(chan struct{})
	go func() { ckv.Set(key, value); close(setDone) }()
	select {
	case <-setDone:
	case <-time.After(2 * time.Second):
	}

	// (3) The simulation read resumes and caches whatever it saw.
	close(gated.readResume)
	<-simulateDone
	<-setDone

	// The committed value is on disk. This is exactly what the raw
	// historical query against the halted node showed.
	require.Equal(t, value, underlying.Get(key),
		"underlying store must contain the committed value")

	// (4) The next block's execution must see the committed value. On the
	// unsynchronized cache it reads the poisoned nil instead.
	require.Equal(t, value, ckv.Get(key),
		"phantom 'state key not found': cache returned stale nil for a key that is on disk")
}

// TestCommitKVStoreCacheStaleValueResurrection is the mirror image: a
// suspended read caches a stale value over a concurrent Delete, so the node
// keeps seeing state that no longer exists on disk.
func TestCommitKVStoreCacheStaleValueResurrection(t *testing.T) {
	gated, underlying, ckv := newGatedCacheStore(t)

	key := []byte("initial_acc_fees/trade_6329")
	value := []byte("closed_and_deleted")

	// Seed the underlying store directly so the cache has no entry yet.
	underlying.Set(key, value)

	// A simulation read misses the cache, reads the value from the
	// underlying store, and is suspended before caching it.
	gated.armed.Store(true)
	simulateDone := make(chan []byte, 1)
	go func() { simulateDone <- ckv.Get(key) }()
	<-gated.readStarted

	// The block commit deletes the key while the read is suspended.
	deleteDone := make(chan struct{})
	go func() { ckv.Delete(key); close(deleteDone) }()
	select {
	case <-deleteDone:
	case <-time.After(2 * time.Second):
	}

	// The read resumes and caches the stale value.
	close(gated.readResume)
	<-simulateDone
	<-deleteDone

	require.Nil(t, underlying.Get(key),
		"underlying store must not contain the deleted key")
	require.Nil(t, ckv.Get(key),
		"phantom resurrection: cache returned a value that was deleted on disk")
}

// TestCommitKVStoreCacheConcurrentGetSet exercises the production access
// pattern (Simulate reads through the shared cache concurrently with block
// commit write-through) over many keys. `go test -race` reports the data race
// on the underlying store deterministically; without -race the final sweep
// catches whichever keys happened to be poisoned, which is timing-dependent.
func TestCommitKVStoreCacheConcurrentGetSet(t *testing.T) {
	const numKeys = 2000

	db := dbm.NewMemDB()
	tree, err := iavl.NewMutableTree(db, 100, false)
	require.NoError(t, err)
	ckv := cache.NewCommitKVStoreCache(iavlstore.UnsafeNewStore(tree), numKeys)

	keyOf := func(i int) []byte { return []byte(fmt.Sprintf("key_%06d", i)) }
	valueOf := func(i int) []byte { return []byte(fmt.Sprintf("value_%06d", i)) }

	var wg sync.WaitGroup
	commitDone := make(chan struct{})
	wg.Add(2)
	go func() { // block commits writing through the cache
		defer wg.Done()
		defer close(commitDone)
		for i := 0; i < numKeys; i++ {
			ckv.Set(keyOf(i), valueOf(i))
		}
	}()
	go func() { // Simulate traffic reading through the same cache
		defer wg.Done()
		for {
			select {
			case <-commitDone:
				return
			default:
			}
			for i := 0; i < numKeys; i++ {
				ckv.Get(keyOf(i))
			}
		}
	}()
	wg.Wait()

	poisoned := 0
	for i := 0; i < numKeys; i++ {
		if !bytes.Equal(ckv.Get(keyOf(i)), valueOf(i)) {
			poisoned++
		}
	}
	require.Zerof(t, poisoned,
		"%d of %d committed keys read back stale cache entries", poisoned, numKeys)
}
