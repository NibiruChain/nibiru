package evmstate

// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

// JournalChange, also called a "journal entry", is a modification entry in the
// state change journal that can be reverted on demand.
type JournalChange interface {
	// Revert undoes the changes introduced by this journal entry.
	Revert(*SDB)

	// Dirtied returns the Ethereum address modified by this journal entry.
	Dirtied() *common.Address
}

// journal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in the case of an execution
// exception or request for reversal.
type journal struct {
	entries []JournalChange        // Current changes tracked by the journal
	dirties map[common.Address]int // Dirty accounts and the number of changes
}

// newJournal creates a new initialized journal.
func newJournal() *journal {
	return &journal{
		dirties: make(map[common.Address]int),
	}
}

// sortedDirties sort the dirty addresses for deterministic iteration
func (j *journal) sortedDirties() []common.Address {
	keys := make([]common.Address, 0, len(j.dirties))
	for k := range j.dirties {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) < 0
	})
	return keys
}

// append inserts a new modification entry to the end of the change journal.
func (j *journal) append(entry JournalChange) {
	j.entries = append(j.entries, entry)
	if addr := entry.Dirtied(); addr != nil {
		j.dirties[*addr]++
	}
}

// dirty explicitly sets an address to dirty, even if the change entries would
// otherwise suggest it as clean. It is copied directly from go-ethereum. In the
// words of the library authors, "this method is an ugly hack to handle the
// RIPEMD precompile consensus exception." - geth/core/state/journal.go
func (j *journal) dirty(addr common.Address) {
	j.dirties[addr]++
}

// Revert undoes a batch of journalled modifications along with any Reverted
// dirty handling too.
func (j *journal) Revert(statedb *SDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].Revert(statedb)

		// Drop any dirty tracking induced by the change
		if addr := j.entries[i].Dirtied(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// Length returns the current number of entries in the journal.
func (j *journal) Length() int {
	return len(j.entries)
}

// ------------------------------------------------------
// suicideChange

type suicideChange struct {
	account     *common.Address
	prev        bool // whether account had already suicided
	prevbalance *big.Int
}

var _ JournalChange = suicideChange{}

func (ch suicideChange) Revert(s *SDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.SelfDestructed = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

func (ch suicideChange) Dirtied() *common.Address {
	return ch.account
}

// ------------------------------------------------------
// transientStorageChange represents a [JournalChange] for whenver a transient
// storage slot changes.
var _ JournalChange = transientStorageChange{}

// transientStorageChange: [JournalChange] implementation for whenver a transient
// storage slot changes
type transientStorageChange struct {
	address        *common.Address
	key, prevValue common.Hash
}

func (ch transientStorageChange) Revert(s *SDB) {
	s.transientStorage.Set(*ch.address, ch.key, ch.prevValue)
}

func (ch transientStorageChange) Dirtied() *common.Address {
	return nil
}

var _ JournalChange = touchChange{}

// touchChange is a journal entry that marks an account as 'touched'.
//
// This is necessary to comply with EIP-161, which defines that accounts must be
// considered for deletion at the end of a transaction if they remain empty
// (balance, nonce, and code are all zero) and were not accessed during the
// transaction.
//
// Calling 'touch' ensures that the account is retained in state for the duration
// of the transaction, even if it remains empty. This helps prevent unintended
// deletions of accounts that are interacted with but have no effective state
// changes.
//
// No actual state is reverted during a `touchChange.revert()` — its presence in
// the journal is only meaningful for dirtiness tracking and snapshot
// consistency.
type touchChange struct {
	account common.Address
}

// Revert is an intentional no-op. To revert a [touchChange], do nothing.
func (ch touchChange) Revert(s *SDB) {}

func (ch touchChange) Dirtied() *common.Address {
	return &ch.account
}

// createContractChange represents an account becoming a contract-account.
// This event happens prior to executing initcode. The journal-event simply
// manages the created-flag, in order to allow same-tx destruction.
type createContractChange struct {
	account common.Address
}

func (ch createContractChange) Revert(s *SDB) {
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.newContract = false
}

func (ch createContractChange) Dirtied() *common.Address {
	return nil
}
