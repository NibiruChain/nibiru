// Copyright (c) 2023-2024 Nibi, Inc.
package types

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// AddrLocker is a mutex (mutual exclusion lock) structure used to avoid querying
// outdated account data. It prevents data races by allowing only one goroutine
// to access critical sections at a time.
type AddrLocker struct {
	// mu protects access to the locks map
	mu    sync.Mutex
	locks map[common.Address]*sync.Mutex
}

// lock returns the mutex lock of the given Ethereum address. If no mutex exists
// for the address, it creates a new one. This function ensures that each address
// has exactly one mutex associated with it, and it is thread-safe.
//
// The returned mutex is not locked; callers are responsible for locking and
// unlocking it as necessary.
func (l *AddrLocker) lock(address common.Address) *sync.Mutex {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.locks == nil {
		l.locks = make(map[common.Address]*sync.Mutex)
	}
	if _, ok := l.locks[address]; !ok {
		l.locks[address] = new(sync.Mutex)
	}
	return l.locks[address]
}

// LockAddr acquires the mutex for a specific address, blocking if it is already
// held by another goroutine. The mutex lock prevents an identical nonce from
// being read again during the time that the first transaction is being signed.
func (l *AddrLocker) LockAddr(address common.Address) {
	l.lock(address).Lock()
}

// UnlockAddr unlocks the mutex for a specific address, allowing other goroutines
// to acquire it.
func (l *AddrLocker) UnlockAddr(address common.Address) {
	l.lock(address).Unlock()
}
