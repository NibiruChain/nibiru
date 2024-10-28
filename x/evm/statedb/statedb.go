// The "evm/statedb" package implements a go-ethereum [vm.StateDB] with state
// management and journal changes specific to the Nibiru EVM.
//
// This package plays a critical role in managing the state of accounts,
// contracts, and storage while handling atomicity, caching, and state
// modifications. It ensures that state transitions made during the
// execution of smart contracts are either committed or reverted based
// on transaction outcomes.
//
// StateDB structs used to store anything within the state tree, including
// accounts, contracts, and contract storage.
// Note that Nibiru's state tree is an IAVL tree, which differs from the Merkle
// Patricia Trie structure seen on Ethereum mainnet.
//
// StateDBs also take care of caching and handling nested states.
package statedb

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"fmt"
	"math/big"
	"sort"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

var _ vm.StateDB = &StateDB{}

// StateDB structs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	keeper Keeper

	// evmTxCtx is the persistent context used for official `StateDB.Commit` calls.
	evmTxCtx sdk.Context

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	Journal        *journal
	validRevisions []revision
	nextRevisionID int

	stateObjects map[common.Address]*stateObject

	txConfig TxConfig

	// cacheCtx: An sdk.Context produced from the [StateDB.ctx] with the
	// multi-store cached and a new event manager. The cached context
	// (`cacheCtx`) is written to the persistent context (`ctx`) when
	// `writeCacheCtx` is called.
	cacheCtx sdk.Context

	// writeToCommitCtxFromCacheCtx is the "write" function received from
	// `s.evmTxCtx.CacheContext()`. It saves mutations on s.cacheCtx to the StateDB's
	// commit context (s.evmTxCtx). This synchronizes the multistore and event manager
	// of the two contexts.
	writeToCommitCtxFromCacheCtx func()

	// The number of precompiled contract calls within the current transaction
	multistoreCacheCount uint8

	// The refund counter, also used by state transitioning.
	refund uint64

	// Per-transaction logs
	logs []*gethcore.Log

	// Per-transaction access list
	accessList *accessList
}

func FromVM(evmObj *vm.EVM) *StateDB {
	return evmObj.StateDB.(*StateDB)
}

// New creates a new state from a given trie.
func New(ctx sdk.Context, keeper Keeper, txConfig TxConfig) *StateDB {
	fmt.Println("statedb.New called")
	return &StateDB{
		keeper:       keeper,
		evmTxCtx:     ctx,
		stateObjects: make(map[common.Address]*stateObject),
		Journal:      newJournal(),
		accessList:   newAccessList(),

		txConfig: txConfig,
	}
}

// revision is the identifier of a version of state.
// it consists of an auto-increment id and a journal index.
// it's safer to use than using journal index alone.
type revision struct {
	id           int
	journalIndex int
}

// Keeper returns the underlying `Keeper`
func (s *StateDB) Keeper() Keeper {
	return s.keeper
}

// GetEvmTxContext returns the EVM transaction context.
func (s *StateDB) GetEvmTxContext() sdk.Context {
	return s.evmTxCtx
}

// GetCacheContext: Getter for testing purposes.
func (s *StateDB) GetCacheContext() *sdk.Context {
	if s.writeToCommitCtxFromCacheCtx == nil {
		return nil
	}
	return &s.cacheCtx
}

// AddLog adds a log, called by evm.
func (s *StateDB) AddLog(log *gethcore.Log) {
	s.Journal.append(addLogChange{})

	log.TxHash = s.txConfig.TxHash
	log.BlockHash = s.txConfig.BlockHash
	log.TxIndex = s.txConfig.TxIndex
	log.Index = s.txConfig.LogIndex + uint(len(s.logs))
	s.logs = append(s.logs, log)
}

// Logs returns the logs of current transaction.
func (s *StateDB) Logs() []*gethcore.Log {
	return s.logs
}

// AddRefund adds gas to the refund counter
func (s *StateDB) AddRefund(gas uint64) {
	s.Journal.append(refundChange{prev: s.refund})
	s.refund += gas
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (s *StateDB) SubRefund(gas uint64) {
	s.Journal.append(refundChange{prev: s.refund})
	if gas > s.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, s.refund))
	}
	s.refund -= gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (s *StateDB) Exist(addr common.Address) bool {
	return s.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *StateDB) Empty(addr common.Address) bool {
	so := s.getStateObject(addr)
	return so == nil || so.isEmpty()
}

// GetBalance retrieves the balance from the given address or 0 if object not found
func (s *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.Big0
}

// GetNonce returns the nonce of account, 0 if not exists.
func (s *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

// GetCode returns the code of account, nil if not exists.
func (s *StateDB) GetCode(addr common.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code()
	}
	return nil
}

// GetCodeSize returns the code size of account.
func (s *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.CodeSize()
	}
	return 0
}

// GetCodeHash returns the code hash of account.
func (s *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

// GetState retrieves a value from the given account's storage trie.
func (s *StateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(hash)
	}
	return common.Hash{}
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetCommittedState(hash)
	}
	return common.Hash{}
}

// GetRefund returns the current value of the refund counter.
func (s *StateDB) GetRefund() uint64 {
	return s.refund
}

// HasSuicided returns if the contract is suicided in current transaction.
func (s *StateDB) HasSuicided(addr common.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Suicided
	}
	return false
}

// AddPreimage records a SHA3 preimage seen by the VM.
// AddPreimage performs a no-op since the EnablePreimageRecording flag is disabled
// on the vm.Config during state transitions. No store trie preimages are written
// to the database.
func (s *StateDB) AddPreimage(_ common.Hash, _ []byte) {}

// getStateObject retrieves a state object given by the address, returning nil if
// the object is not found.
func (s *StateDB) getStateObject(addr common.Address) *stateObject {
	// Prefer live objects if any is available
	if obj := s.stateObjects[addr]; obj != nil {
		return obj
	}

	// If no live objects are available, load it from keeper
	account := s.keeper.GetAccount(s.evmTxCtx, addr)
	if account == nil {
		return nil
	}

	// Insert into the live set
	obj := newObject(s, addr, *account)
	s.setStateObject(obj)
	return obj
}

// getOrNewStateObject retrieves a state object or create a new state object if nil.
func (s *StateDB) getOrNewStateObject(addr common.Address) *stateObject {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject, _ = s.createObject(addr)
	}
	return stateObject
}

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (s *StateDB) createObject(addr common.Address) (newobj, prev *stateObject) {
	prev = s.getStateObject(addr)

	newobj = newObject(s, addr, Account{})
	if prev == nil {
		s.Journal.append(createObjectChange{account: &addr})
	} else {
		s.Journal.append(resetObjectChange{prev: prev})
	}
	s.setStateObject(newobj)
	if prev != nil {
		return newobj, prev
	}
	return newobj, nil
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
// 1. sends funds to sha(account ++ (nonce + 1))
// 2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (s *StateDB) CreateAccount(addr common.Address) {
	newObj, prev := s.createObject(addr)
	if prev != nil {
		newObj.setBalance(prev.account.BalanceWei)
	}
}

// ForEachStorage iterate the contract storage, the iteration order is not defined.
func (s *StateDB) ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) error {
	so := s.getStateObject(addr)
	if so == nil {
		return nil
	}
	s.keeper.ForEachStorage(s.evmTxCtx, addr, func(key, value common.Hash) bool {
		if value, dirty := so.DirtyStorage[key]; dirty {
			return cb(key, value)
		}
		if len(value) > 0 {
			return cb(key, value)
		}
		return true
	})
	return nil
}

func (s *StateDB) setStateObject(object *stateObject) {
	s.stateObjects[object.Address()] = object
}

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr.
func (s *StateDB) AddBalance(addr common.Address, wei *big.Int) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(wei)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (s *StateDB) SubBalance(addr common.Address, wei *big.Int) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(wei)
	}
}

func (s *StateDB) SetBalanceWei(addr common.Address, wei *big.Int) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(wei)
	}
}

// SetNonce sets the nonce of account.
func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

// SetCode sets the code of account.
func (s *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

// SetState sets the contract state.
func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, value)
	}
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after Suicide.
func (s *StateDB) Suicide(addr common.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return false
	}
	s.Journal.append(suicideChange{
		account:     &addr,
		prev:        stateObject.Suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.Suicided = true
	stateObject.account.BalanceWei = new(big.Int)

	return true
}

// PrepareAccessList handles the preparatory steps for executing a state transition with
// regards to both EIP-2929 and EIP-2930:
//
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// This method should only be called if Yolov3/Berlin/2929+2930 is applicable at the current number.
func (s *StateDB) PrepareAccessList(
	sender common.Address,
	dst *common.Address,
	precompiles []common.Address,
	list gethcore.AccessList,
) {
	s.AddAddressToAccessList(sender)
	if dst != nil {
		s.AddAddressToAccessList(*dst)
		// If it's a create-tx, the destination will be added inside evm.create
	}
	for _, addr := range precompiles {
		s.AddAddressToAccessList(addr)
	}
	for _, el := range list {
		s.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			s.AddSlotToAccessList(el.Address, key)
		}
	}
}

// AddAddressToAccessList adds the given address to the access list
func (s *StateDB) AddAddressToAccessList(addr common.Address) {
	if s.accessList.AddAddress(addr) {
		s.Journal.append(accessListAddAccountChange{&addr})
	}
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (s *StateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	addrMod, slotMod := s.accessList.AddSlot(addr, slot)
	if addrMod {
		// In practice, this should not happen, since there is no way to enter the
		// scope of 'address' without having the 'address' become already added
		// to the access list (via call-variant, create, etc).
		// Better safe than sorry, though
		s.Journal.append(accessListAddAccountChange{&addr})
	}
	if slotMod {
		s.Journal.append(accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *StateDB) AddressInAccessList(addr common.Address) bool {
	return s.accessList.ContainsAddress(addr)
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return s.accessList.Contains(addr, slot)
}

// Snapshot returns an identifier for the current revision of the state.
func (s *StateDB) Snapshot() int {
	id := s.nextRevisionID
	s.nextRevisionID++
	s.validRevisions = append(s.validRevisions, revision{id, s.Journal.Length()})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(s.validRevisions), func(i int) bool {
		return s.validRevisions[i].id >= revid
	})
	if idx == len(s.validRevisions) || s.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := s.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	s.Journal.Revert(s, snapshot)
	s.validRevisions = s.validRevisions[:idx]
}

// errorf: wrapper of "fmt.Errorf" specific to the current Go package.
func errorf(format string, args ...any) error {
	return fmt.Errorf("StateDB error: "+format, args...)
}

// Commit writes the dirty journal state changes to the EVM Keeper. The
// StateDB object cannot be reused after [Commit] has completed. A new
// object needs to be created from the EVM.
//
// cacheCtxSyncNeeded: If one of the [Nibiru-Specific Precompiled Contracts] was
// called, a [JournalChange] of type [PrecompileSnapshotBeforeRun] gets added and
// we branch off a cache of the commit context (s.evmTxCtx).
//
// [Nibiru-Specific Precompiled Contracts]: https://nibiru.fi/docs/evm/precompiles/nibiru.html
func (s *StateDB) Commit() error {
	if s.writeToCommitCtxFromCacheCtx != nil {
		s.writeToCommitCtxFromCacheCtx()
	}
	return s.commitCtx(s.GetEvmTxContext())
}

// CommitCacheCtx is identical to [StateDB.Commit], except it:
// (1) uses the cacheCtx of the [StateDB] and
// (2) does not save mutations of the cacheCtx to the commit context (s.evmTxCtx).
// The reason for (2) is that the overall EVM transaction (block, not internal)
// is only finalized when [Commit] is called, not when [CommitCacheCtx] is
// called.
func (s *StateDB) CommitCacheCtx() error {
	return s.commitCtx(s.cacheCtx)
}

// commitCtx writes the dirty journal state changes to the EVM Keeper. The
// StateDB object cannot be reused after [commitCtx] has completed. A new
// object needs to be created from the EVM.
func (s *StateDB) commitCtx(ctx sdk.Context) error {
	for _, addr := range s.Journal.sortedDirties() {
		obj := s.getStateObject(addr)
		if obj == nil {
			s.Journal.dirties[addr] = 0
			continue
		}
		if obj.Suicided {
			// Invariant: After [StateDB.Suicide] for some address, the
			// corresponding account's state object is only available until the
			// state is committed.
			if err := s.keeper.DeleteAccount(ctx, obj.Address()); err != nil {
				return errorf("failed to delete account: %w", err)
			}
			delete(s.stateObjects, addr)
		} else {
			if obj.code != nil && obj.DirtyCode {
				s.keeper.SetCode(ctx, obj.CodeHash(), obj.code)
			}
			if err := s.keeper.SetAccount(ctx, obj.Address(), obj.account.ToNative()); err != nil {
				return errorf("failed to set account: %w", err)
			}
			for _, key := range obj.DirtyStorage.SortedKeys() {
				dirtyVal := obj.DirtyStorage[key]
				// Values that match origin storage are not dirty.
				if dirtyVal == obj.OriginStorage[key] {
					continue
				}
				// Persist committed changes
				s.keeper.SetState(ctx, obj.Address(), key, dirtyVal.Bytes())
				obj.OriginStorage[key] = dirtyVal
			}
		}
		// TODO: UD-DEBUG: Assume clean to pretend for tests
		// Reset the dirty count to 0 because all state changes for this dirtied
		// address in the journal have been committed.
		s.Journal.dirties[addr] = 0
	}
	return nil
}

func (s *StateDB) CacheCtxForPrecompile(precompileAddr common.Address) (
	sdk.Context, PrecompileCalled,
) {
	if s.writeToCommitCtxFromCacheCtx == nil {
		s.cacheCtx, s.writeToCommitCtxFromCacheCtx = s.evmTxCtx.CacheContext()
	}
	return s.cacheCtx, PrecompileCalled{
		MultiStore: s.cacheCtx.MultiStore().(store.CacheMultiStore).Copy(),
		Events:     s.cacheCtx.EventManager().Events(),
	}
}

// SavePrecompileCalledJournalChange adds a snapshot of the commit multistore
// ([PrecompileCalled]) to the [StateDB] journal at the end of
// successful invocation of a precompiled contract. This is necessary to revert
// intermediate states where an EVM contract augments the multistore with a
// precompile and an inconsistency occurs between the EVM module and other
// modules.
//
// See [PrecompileCalled] for more info.
func (s *StateDB) SavePrecompileCalledJournalChange(
	precompileAddr common.Address,
	journalChange PrecompileCalled,
) error {
	s.Journal.append(journalChange)
	s.multistoreCacheCount++
	if s.multistoreCacheCount > maxMultistoreCacheCount {
		return fmt.Errorf(
			"exceeded maximum number Nibiru-specific precompiled contract calls in one transaction (%d). Called address %s",
			maxMultistoreCacheCount, precompileAddr.Hex(),
		)
	}
	return nil
}

const maxMultistoreCacheCount uint8 = 10

// StateObjects: Returns a copy of the [StateDB.stateObjects] map.
func (s *StateDB) StateObjects() map[common.Address]*stateObject {
	copyOfMap := make(map[common.Address]*stateObject)
	for key, val := range s.stateObjects {
		copyOfMap[key] = val
	}
	return copyOfMap
}
