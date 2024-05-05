package rpc_test

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	rpc "github.com/NibiruChain/nibiru/eth/rpc"
)

type SuiteAddrLocker struct {
	suite.Suite
}

func TestSuiteAddrLocker(t *testing.T) {
	suite.Run(t, new(SuiteAddrLocker))
}

// TestLockAddr: This test checks that the lock mechanism prevents multiple
// goroutines from entering critical sections of code simultaneously for the same
// address.
func (s *SuiteAddrLocker) TestLockAddr() {
	// Setup: Lock the address
	locker := &rpc.AddrLocker{}
	addr := common.HexToAddress("0x123")
	locker.LockAddr(addr)

	// Concurrent Lock Attempt: Attempt to lock again in a separate goroutine. If
	// the initial lock is effective, this attempt should block and not complete
	// immediately.
	done := make(chan bool)
	go func() {
		locker.LockAddr(addr) // This should block if the first lock is effective
		done <- true
	}()

	// Assertion: A select statement is used to check if the channel receives a
	// value, which would indicate the lock did not block as expected.
	select {
	case <-done:
		s.Fail("LockAddr did not block the second call as expected")
	default:
		// expected behavior, continue test
	}

	// Cleanup: Unlock and allow the goroutine to proceed
	locker.UnlockAddr(addr)
	<-done // Ensure goroutine completes
}

func (s *SuiteAddrLocker) TestUnlockAddr() {
	// Setup: Lock the address
	locker := &rpc.AddrLocker{}
	addr := common.HexToAddress("0x123")
	locker.LockAddr(addr)

	locker.UnlockAddr(addr)

	// Try re-locking to test if unlock was successful
	locked := make(chan bool)
	go func() {
		locker.LockAddr(addr) // This should not block if unlock worked
		locked <- true
		locker.UnlockAddr(addr)
	}()

	select {
	case <-locked:
		// expected behavior, continue
	case <-time.After(time.Second):
		s.Fail("UnlockAddr did not effectively unlock the mutex")
	}
}

func (s *SuiteAddrLocker) TestMultipleAddresses() {
	locker := &rpc.AddrLocker{}
	addr1 := common.HexToAddress("0x123")
	addr2 := common.HexToAddress("0x456")

	locker.LockAddr(addr1)
	locked := make(chan bool)

	go func() {
		locker.LockAddr(addr2) // This should not block if locks are address-specific
		locked <- true
		locker.UnlockAddr(addr2)
	}()

	select {
	case <-locked:
		// expected behavior, continue
	case <-time.After(time.Second):
		s.Fail("Locks are not address-specific as expected")
	}

	locker.UnlockAddr(addr1)
}

// TestConcurrentAccess: Tests the system's behavior under high concurrency,
// specifically ensuring that the lock can handle multiple locking and unlocking
// operations on the same address without leading to race conditions or
// deadlocks.
func (s *SuiteAddrLocker) TestConcurrentAccess() {
	locker := &rpc.AddrLocker{}
	addr := common.HexToAddress("0x789")
	var wg sync.WaitGroup

	// Spawn 100 goroutines, each locking and unlocking the same address.
	// Each routine will hod the lock briefly to simulate work done during the
	// lock (like an Ethereum query).
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			locker.LockAddr(addr)
			time.Sleep(time.Millisecond * 5) // Simulate work
			locker.UnlockAddr(addr)
			wg.Done()
		}()
	}

	// Cleanup: Wait for all goroutines to complete
	wg.Wait()
}
