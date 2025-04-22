package statedb

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

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// JournalChange, also called a "journal entry", is a modification entry in the
// state change journal that can be reverted on demand.
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

// dirty explicitly sets an address to dirty, even if the change entries would
// otherwise suggest it as clean. It is copied directly from go-ethereum. In the
// words of the library authors, "this method is an ugly hack to handle the
// RIPEMD precompile consensus exception." - geth/core/state/journal.go
func (j *journal) dirty(addr common.Address) {
	j.dirties[addr]++
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

// Length returns the current number of entries in the journal.
func (j *journal) Length() int {
	return len(j.entries)
}

// ------------------------------------------------------
// createObjectChange

// createObjectChange: [JournalChange] implementation for when
// a new account (called an "object" in this context) is created in state.
type createObjectChange struct {
	account *common.Address
}

var _ JournalChange = createObjectChange{}

func (ch createObjectChange) Revert(s *StateDB) {
	delete(s.stateObjects, *ch.account)
}

func (ch createObjectChange) Dirtied() *common.Address {
	return ch.account
}

// ------------------------------------------------------
// resetObjectChange

// resetObjectChange: [JournalChange] for an account that needs its
// original state reset. This is used when an account's state is being replaced
// and we need to revert to the previous version.
type resetObjectChange struct {
	prev *stateObject
}

var _ JournalChange = resetObjectChange{}

func (ch resetObjectChange) Revert(s *StateDB) {
	s.setStateObject(ch.prev)
}

func (ch resetObjectChange) Dirtied() *common.Address {
	return nil
}

// ------------------------------------------------------
// suicideChange

type suicideChange struct {
	account     *common.Address
	prev        bool // whether account had already suicided
	prevbalance *big.Int
}

var _ JournalChange = suicideChange{}

func (ch suicideChange) Revert(s *StateDB) {
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
// balanceChange

// balanceChange: [JournalChange] for an update to the wei balance of an account.
type balanceChange struct {
	account *common.Address
	prevWei *uint256.Int
}

var _ JournalChange = balanceChange{}

func (ch balanceChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setBalance(ch.prevWei.ToBig())
}

func (ch balanceChange) Dirtied() *common.Address {
	return ch.account
}

// ------------------------------------------------------
// nonceChange

// nonceChange: [JournalChange] for an update to the nonce of an account.
// The nonce is a counter of the number of transactions sent from an account.
type nonceChange struct {
	account *common.Address
	prev    uint64
}

var _ JournalChange = nonceChange{}

func (ch nonceChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setNonce(ch.prev)
}

func (ch nonceChange) Dirtied() *common.Address {
	return ch.account
}

// ------------------------------------------------------
// codeChange

// codeChange: [JournalChange] for an update to an account's code (smart contract
// bytecode). The previous code and hash for the code are stored to enable
// reversion.
type codeChange struct {
	account            *common.Address
	prevcode, prevhash []byte
}

var _ JournalChange = codeChange{}

func (ch codeChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch codeChange) Dirtied() *common.Address {
	return ch.account
}

// ------------------------------------------------------
// storageChange

// storageChange: [JournalChange] for the modification of a single key and value
// within a contract's storage.
type storageChange struct {
	account       *common.Address
	key, prevalue common.Hash
	origin        common.Hash
}

var _ JournalChange = storageChange{}

func (ch storageChange) Revert(s *StateDB) {
	s.getStateObject(*ch.account).setState(ch.key, ch.prevalue, ch.origin)
}

func (ch storageChange) Dirtied() *common.Address {
	return ch.account
}

// ------------------------------------------------------
// refundChange

// refundChange: [JournalChange] for the global gas refund counter.
// This tracks changes to the gas refund value during contract execution.
type refundChange struct {
	prev uint64
}

var _ JournalChange = refundChange{}

func (ch refundChange) Revert(s *StateDB) {
	s.refund = ch.prev
}

func (ch refundChange) Dirtied() *common.Address {
	return nil
}

// ------------------------------------------------------
// addLogChange

// addLogChange represents [JournalChange] for a new log addition.
// When reverted, it removes the last log from the accumulated logs list.
type addLogChange struct{}

var _ JournalChange = addLogChange{}

func (ch addLogChange) Revert(s *StateDB) {
	s.logs = s.logs[:len(s.logs)-1]
}

func (ch addLogChange) Dirtied() *common.Address {
	return nil
}

// ------------------------------------------------------
// accessListAddAccountChange

// accessListAddAccountChange represents [JournalChange] for when an address
// is added to the access list. Access lists track warm storage slots for
// gas cost calculations.
type accessListAddAccountChange struct {
	address *common.Address
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

// ------------------------------------------------------
// accessListAddSlotChange

// accessListAddSlotChange: [JournalChange] implementations for
type accessListAddSlotChange struct {
	address *common.Address
	slot    *common.Hash
}

// accessListAddSlotChange represents a [JournalChange] for when a storage slot
// is added to an address's access list entry. This tracks individual storage
// slots that have been accessed.
var _ JournalChange = accessListAddSlotChange{}

func (ch accessListAddSlotChange) Revert(s *StateDB) {
	s.accessList.DeleteSlot(*ch.address, *ch.slot)
}

func (ch accessListAddSlotChange) Dirtied() *common.Address {
	return nil
}

// ------------------------------------------------------
// PrecompileSnapshotBeforeRun

// PrecompileCalled: Precompiles can alter persistent storage of other
// modules. These changes to persistent storage are not reverted by a `Revert` of
// [JournalChange] by default, as it generally manages only changes to accounts
// and Bank balances for ether (NIBI).
//
// As a workaround to make state changes from precompiles reversible, we store
// [PrecompileCalled] snapshots that sync and record the prior state
// of the other modules, allowing precompile calls to truly be reverted.
//
// As a simple example, suppose that a transaction calls a precompile.
//  1. If the precompile changes the state in the Bank Module or Wasm module
//  2. The call gets reverted (`revert()` in Solidity), which shoud restore the
//     state to a in-memory snapshot recorded on the StateDB journal.
//  3. This could cause a problem where changes to the rest of the blockchain state
//     are still in effect following the reversion in the EVM state DB.
type PrecompileCalled struct {
	MultiStore store.CacheMultiStore
	Events     sdk.Events
}

var _ JournalChange = PrecompileCalled{}

// Revert rolls back the [StateDB] cache context to the state it was in prior to
// the precompile call. Modifications to this cache context are pushed to the
// commit context (s.evmTxCtx) when [StateDB.Commit] is executed.
func (ch PrecompileCalled) Revert(s *StateDB) {
	s.cacheCtx = s.cacheCtx.WithMultiStore(ch.MultiStore)
	// Rewrite the `writeCacheCtxFn` using the same logic as sdk.Context.CacheCtx
	s.writeToCommitCtxFromCacheCtx = func() {
		s.evmTxCtx.EventManager().EmitEvents(ch.Events)
		// TODO: Check correctness of the emitted events
		// https://github.com/NibiruChain/nibiru/issues/2096
		ch.MultiStore.Write()
	}
}

func (ch PrecompileCalled) Dirtied() *common.Address {
	return nil
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

func (ch transientStorageChange) Revert(s *StateDB) {
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
func (ch touchChange) Revert(s *StateDB) {}

func (ch touchChange) Dirtied() *common.Address {
	return &ch.account
}

// createContractChange represents an account becoming a contract-account.
// This event happens prior to executing initcode. The journal-event simply
// manages the created-flag, in order to allow same-tx destruction.
type createContractChange struct {
	account common.Address
}

func (ch createContractChange) Revert(s *StateDB) {
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.newContract = false
}

func (ch createContractChange) Dirtied() *common.Address {
	return nil
}
