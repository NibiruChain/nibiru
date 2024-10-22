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

package statedb

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// JournalChange is a modification entry in the state change journal that can be
// Reverted on demand.
type JournalChange interface {
	// Revert undoes the changes introduced by this journal entry.
	Revert(*StateDB)

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

// Revert undoes a batch of journalled modifications along with any Reverted
// dirty handling too.
func (j *journal) Revert(statedb *StateDB, snapshot int) {
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

// length returns the current number of entries in the journal.
func (j *journal) length() int {
	return len(j.entries)
}

type (
	// createObjectChange: [JournalChange] implementation for when
	// a new account (called an "object" in this context) is created in state.
	createObjectChange struct {
		account *common.Address
	}
	// resetObjectChange: [JournalChange] implementation for when
	// a backup for a state object is needed in order to revert state changes
	// that overwrite an existing object.
	resetObjectChange struct {
		prev *stateObject
	}
	suicideChange struct {
		account     *common.Address
		prev        bool // whether account had already suicided
		prevbalance *big.Int
	}

	// balanceChange: [JournalChange] for an update to an account's wei balance
	balanceChange struct {
		account *common.Address
		prevWei *big.Int
	}
	// nonceChange: [JournalChange] for an update to an account's nonce.
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	storageChange struct {
		account       *common.Address
		key, prevalue common.Hash
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct{}

	// Changes to the access list
	accessListAddAccountChange struct {
		address *common.Address
	}
	// accessListAddSlotChange: [JournalChange] implementations for
	accessListAddSlotChange struct {
		address *common.Address
		slot    *common.Hash
	}
)

// ------------------------------------------------------
// PrecompileSnapshotBeforeRun

var _ JournalChange = PrecompileSnapshotBeforeRun{}

type PrecompileSnapshotBeforeRun struct {
	MultiStore store.CacheMultiStore
	Events     sdk.Events
}

func (ch PrecompileSnapshotBeforeRun) Revert(s *StateDB) {
	// TODO: Revert PrecompileSnapshotBeforeRun
	// Restore the multistore recorded in the journal entry

	fmt.Printf("TODO: UD-DEBUG: PrecompileSnapshotBeforeRun.Revert called\n")

	s.cacheCtx = s.cacheCtx.WithMultiStore(ch.MultiStore)
	// Rewrite the `writeCacheCtxFn` using the same logic as sdk.Context.CacheCtx
	s.writeCacheCtxFn = func() {
		s.ctx.EventManager().EmitEvents(ch.Events)
		ch.MultiStore.Write()
	}
}

func (ch PrecompileSnapshotBeforeRun) Dirtied() *common.Address {
	return nil
}

var (
	_ JournalChange = accessListAddSlotChange{}
	_ JournalChange = createObjectChange{}
	_ JournalChange = resetObjectChange{}
	_ JournalChange = suicideChange{}
	_ JournalChange = balanceChange{}
	_ JournalChange = nonceChange{}
	_ JournalChange = storageChange{}
	_ JournalChange = codeChange{}
	_ JournalChange = refundChange{}
	_ JournalChange = addLogChange{}
	_ JournalChange = accessListAddAccountChange{}
)

func (ch createObjectChange) Revert(s *StateDB) {
	delete(s.stateObjects, *ch.account)
}

func (ch createObjectChange) Dirtied() *common.Address {
	return ch.account
}

func (ch resetObjectChange) Revert(s *StateDB) {
	s.setStateObject(ch.prev)
}

func (ch resetObjectChange) Dirtied() *common.Address {
	return nil
}

func (ch suicideChange) Revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

func (ch suicideChange) Dirtied() *common.Address {
	return ch.account
}

func (ch balanceChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setBalance(ch.prevWei)
}

func (ch balanceChange) Dirtied() *common.Address {
	return ch.account
}

func (ch nonceChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setNonce(ch.prev)
}

func (ch nonceChange) Dirtied() *common.Address {
	return ch.account
}

func (ch codeChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch codeChange) Dirtied() *common.Address {
	return ch.account
}

func (ch storageChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setState(ch.key, ch.prevalue)
}

func (ch storageChange) Dirtied() *common.Address {
	return ch.account
}

func (ch refundChange) Revert(s *StateDB) {
	s.refund = ch.prev
}

func (ch refundChange) Dirtied() *common.Address {
	return nil
}

func (ch addLogChange) Revert(s *StateDB) {
	s.logs = s.logs[:len(s.logs)-1]
}

func (ch addLogChange) Dirtied() *common.Address {
	return nil
}

// When an (address, slot) combination is added, it always results in two
// journal entries if the address is not already present:
//  1. `accessListAddAccountChange`: a journal change for the address
//  2. `accessListAddSlotChange`: a journal change for the (address, slot)
//     combination.
//
// Thus, when reverting, we can safely delete the address, as no storage slots
// remain once the address entry is reverted.
func (ch accessListAddAccountChange) Revert(s *StateDB) {
	s.accessList.DeleteAddress(*ch.address)
}

func (ch accessListAddAccountChange) Dirtied() *common.Address {
	return nil
}

func (ch accessListAddSlotChange) Revert(s *StateDB) {
	s.accessList.DeleteSlot(*ch.address, *ch.slot)
}

func (ch accessListAddSlotChange) Dirtied() *common.Address {
	return nil
}
