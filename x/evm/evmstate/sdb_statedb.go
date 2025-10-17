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
package evmstate

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var _ vm.StateDB = &SDB{}

// SDB structs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type SDB struct {
	keeper *Keeper

	// evmTxCtx is the current context for the EVM transaction. It manages
	// MultiVM state and is safe to modify because it only writes changes to  the
	// root context (the one that created the [SDB]) when [SDB.Commit] is called.
	evmTxCtx sdk.Context

	// TODO: UD-DEBUG: Docs needed.
	// This is the backbone of [SDB.Snapshot] and [SDB.RevertToSnapshot].
	// Optimizes performance by minimizing direct access to the underlying
	// storage for uncommitted mutations produced by the [SDB].
	localState *LocalState
	// This is the backbone of [SDB.Snapshot] and [SDB.RevertToSnapshot].
	savedStates []*LocalState
	// This is the backbone of [SDB.Snapshot] and [SDB.RevertToSnapshot].
	savedCtxs []sdk.Context

	txConfig TxConfig
}

func FromVM(evmObj *vm.EVM) *SDB {
	return evmObj.StateDB.(*SDB)
}

// NewSDB creates a new state from a given trie.
func NewSDB(ctx sdk.Context, k *Keeper, txConfig TxConfig) *SDB {
	sdb := &SDB{
		keeper:     k,
		evmTxCtx:   ctx,
		localState: NewLocalState(), // TODO: UD-DEBUG: new local state
		savedCtxs:  []sdk.Context{},
		txConfig:   txConfig,
	}
	// Initial snapshot is required to guarantee that `RevertToSnapshot(0)` is
	// possible
	sdb.Snapshot()
	return sdb
}

func (k *Keeper) NewSDB(
	ctx sdk.Context, txConfig TxConfig,
) *SDB {
	return NewSDB(ctx, k, txConfig)
}

func (s SDB) TxCfg() TxConfig {
	return s.txConfig
}

// Prepare handles the preparatory steps for executing a state transition with.
// This method must be invoked before state transition.
//
// Berlin fork:
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// EIPs Included:
// - Reset access list (Berlin)
// - Add coinbase to access list (EIP-3651) | Shanghai
// - Reset transient storage (EIP-1153)
func (s *SDB) Prepare(
	_ gethparams.Rules, // only relevant prior to Shangai and Berlin upgrades
	sender, coinbase gethcommon.Address,
	dest *gethcommon.Address,
	precompiles []gethcommon.Address,
	txAccesses gethcore.AccessList,
) {
	s.AddAddressToAccessList(sender)
	if dest != nil {
		s.AddAddressToAccessList(*dest)
		// If it's a create-tx, the destination will be added inside evm.create
	}
	for _, addr := range precompiles {
		s.AddAddressToAccessList(addr)
	}
	for _, el := range txAccesses {
		s.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			s.AddSlotToAccessList(el.Address, key)
		}
	}
	s.AddAddressToAccessList(coinbase) // Shaghai: EIP-3651: warm coinbse

	// EIP-1153: Reset transient storage for beginning of tx execution
	// See core/state/statedb.go from geth.
	s.localState.ContractStorage = make(transientStorage)
}

// Keeper returns the underlying `Keeper`
func (s *SDB) Keeper() *Keeper {
	return s.keeper
}

// Ctx returns the EVM transaction context.
func (s *SDB) Ctx() sdk.Context {
	return s.evmTxCtx
}

// SetCtx overwrites the EVM transaction context.
func (s *SDB) SetCtx(ctx sdk.Context) {
	s.evmTxCtx = ctx
}

// AddLog adds to the EVM's event log for the current transaction.
// [AddLog] uses the [TxConfig] to populate the tx hash, block hash, tx index,
// and event log index.
func (s *SDB) AddLog(ethLog *gethcore.Log) {
	ethLog.TxHash = s.txConfig.TxHash
	ethLog.BlockHash = s.txConfig.BlockHash
	ethLog.TxIndex = s.txConfig.TxIndex
	ethLog.Index = s.txConfig.LogIndex + uint(len(s.Logs()))
	s.localState.logs = append(s.localState.logs, ethLog)
}

// Logs returns the per-transaction event logs.
func (s *SDB) Logs() (allLogs []*gethcore.Log) {
	for _, ls := range append(s.savedStates, s.localState) {
		allLogs = append(allLogs, ls.logs...)
	}
	return allLogs
}

func (s *SDB) LogsJson() (jsonBz []byte) {
	jsonBz, _ = json.MarshalIndent(s.Logs(), "", "  ")
	return
}

// GetRefund returns the current value of the refund counter.
func (s *SDB) GetRefund() uint64 {
	gasRefundBz := func() []byte {
		if len(s.localState.gasRefund) > 0 {
			return s.localState.gasRefund
		}
		for i := len(s.savedStates) - 1; i >= 0; i-- {
			bz := s.savedStates[i].gasRefund
			if len(bz) > 0 {
				return bz
			}
		}
		return nil
	}()
	if gasRefundBz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(gasRefundBz)
}

// AddRefund adds gas to the refund counter
func (s *SDB) AddRefund(gas uint64) {
	newGasRefundBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newGasRefundBz, s.GetRefund()+gas)
	s.localState.gasRefund = newGasRefundBz
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (s *SDB) SubRefund(gas uint64) {
	gasRefund := s.GetRefund()
	if gas > gasRefund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, gasRefund))
	}
	newGasRefundBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newGasRefundBz, gasRefund-gas)
	s.localState.gasRefund = newGasRefundBz
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (s *SDB) Exist(addr gethcommon.Address) bool {
	acc := s.keeper.GetAccount(s.evmTxCtx, addr)
	if acc != nil || s.HasSelfDestructed(addr) {
		return true
	}
	return false
	// Old impl
	// return s.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *SDB) Empty(addr gethcommon.Address) bool {
	// EIP-161: empty iff (nonce == 0) && (balance == 0) && (code == empty)
	acc := s.keeper.GetAccount(s.evmTxCtx, addr)
	if acc == nil {
		return true
	}
	// Nonce
	if acc.Nonce != 0 {
		return false
	}
	// Balance
	if acc.BalanceNwei != nil && !acc.BalanceNwei.IsZero() {
		return false
	}
	// Code hash empty check
	if len(acc.CodeHash) == 0 || bytes.Equal(acc.CodeHash, evm.EmptyCodeHashBz) {
		return true
	}
	return false
}

// GetBalance retrieves the balance from the given address or 0 if object not found
// This function implements the [vm.StateDB] interface.
func (s *SDB) GetBalance(addr gethcommon.Address) *uint256.Int {
	addrBech32 := eth.EthAddrToNibiruAddr(addr)
	return s.keeper.BK().GetWeiBalance(s.evmTxCtx, addrBech32)
}

// GetNonce returns the nonce of account, 0 if not exists.
// This function implements the [vm.StateDB] interface.
func (s *SDB) GetNonce(addr gethcommon.Address) uint64 {
	return s.keeper.GetAccNonce(s.evmTxCtx, addr)
}

// GetCode returns the code of account, nil if not exists.
func (s *SDB) GetCode(addr gethcommon.Address) []byte {
	return s.keeper.GetCode(s.evmTxCtx, s.GetCodeHash(addr))
}

// GetCodeSize returns the code size of account.
func (s *SDB) GetCodeSize(addr gethcommon.Address) int {
	acc := s.keeper.GetAccount(s.evmTxCtx, addr)
	if acc == nil || len(acc.CodeHash) == 0 || bytes.Equal(acc.CodeHash, evm.EmptyCodeHashBz) {
		return 0
	}
	return len(s.GetCode(addr))
}

// GetCodeHash returns the code hash of account.
func (s *SDB) GetCodeHash(addr gethcommon.Address) (codeHash gethcommon.Hash) {
	acc := s.keeper.GetAccount(s.evmTxCtx, addr)
	if acc == nil {
		// Non-existent account → zero hash
		return gethcommon.Hash{}
	}
	if len(acc.CodeHash) == 0 || bytes.Equal(acc.CodeHash, evm.EmptyCodeHashBz) {
		// Existing account but no code → empty code hash
		return gethcommon.Hash(evm.EmptyCodeHashBz)
	}
	return gethcommon.BytesToHash(acc.CodeHash)
}

// GetState retrieves a value from the given account's storage trie.
func (s *SDB) GetState(
	addr gethcommon.Address,
	slotKey gethcommon.Hash,
) (stateValue gethcommon.Hash) {
	return s.keeper.GetState(s.evmTxCtx, addr, slotKey)
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (s *SDB) GetCommittedState(addr gethcommon.Address, hash gethcommon.Hash) gethcommon.Hash {
	return s.keeper.GetState(s.savedCtxs[0], addr, hash)
}

// HasSuicided returns if the contract is suicided in current transaction.
func (s *SDB) HasSuicided(addr gethcommon.Address) bool {
	return s.HasSelfDestructed(addr)
}

// AddPreimage records a SHA3 preimage seen by the VM.
// AddPreimage performs a no-op since the EnablePreimageRecording flag is disabled
// on the vm.Config during state transitions. No store trie preimages are written
// to the database.
func (s *SDB) AddPreimage(_ gethcommon.Hash, _ []byte) {}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
// FIXME: TODO: Check that account balances are meant to be preserved across account reset.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
// 1. sends funds to sha(account ++ (nonce + 1))
// 2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (s *SDB) CreateAccount(addr gethcommon.Address) {
	// Clear balance if there was one for the account
	accBal := s.GetBalance(addr)
	// if !accBal.IsZero() {
	// 	err := s.keeper.SendWei(s.evmTxCtx, addr, evm.EVM_MODULE_ADDRESS, accBal)
	// 	if err != nil {
	// 		panic(err) // TODO: UD-DEBUG: error msg
	// 	}
	// }

	// Create new account or reset an existing one.
	acc := s.keeper.GetAccount(s.evmTxCtx, addr)
	if acc == nil {
		acc = NewEmptyAccount()
	}
	acc.BalanceNwei = accBal
	err := s.keeper.SetAccount(s.evmTxCtx, addr, *acc)
	if err != nil {
		panic(sdbErrorf("%w", err))
	}
	s.localState.AccountChangeMap[addr] = SNAPSHOT_ACC_STATUS_CREATE
}

// CreateContract is used whenever a contract is created. This may be preceded
// by CreateAccount, but that is not required if it already existed in the
// state due to funds sent beforehand.
// This operation sets the 'newContract'-flag, which is required in order to
// correctly handle EIP-6780 'delete-in-same-transaction' logic.
func (s *SDB) CreateContract(addr gethcommon.Address) {
	s.CreateAccount(addr)
	s.localState.AccountChangeMap[addr] = SNAPSHOT_ACC_STATUS_CREATE
}

// TODO: Handle surplus

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr.
// It is used to add funds to the destination account of a transfer.
func (s *SDB) AddBalance(
	addr gethcommon.Address,
	wei *uint256.Int,
	reason tracing.BalanceChangeReason,
) (prevWei uint256.Int) {
	prevWei = *s.GetBalance(addr)
	if wei.Sign() == 0 {
		return
	}
	addrBech32 := eth.EthAddrToNibiruAddr(addr)
	s.keeper.BK().AddWei(s.evmTxCtx, addrBech32, wei)
	// TODO: add sdb tracing logger?
	return
}

// TODO: feat: flag needed to mute events?

// SubBalance subtracts amount from the account associated with addr.
// It is used to remove funds from the origin account of a transfer.
func (s *SDB) SubBalance(
	addr gethcommon.Address,
	wei *uint256.Int,
	reason tracing.BalanceChangeReason,
) (prevWei uint256.Int) {
	prevWei = *s.GetBalance(addr)
	if wei.Sign() == 0 {
		return
	}

	addrBech32 := eth.EthAddrToNibiruAddr(addr)
	err := s.keeper.BK().SubWei(s.evmTxCtx, addrBech32, wei)
	if err != nil {
		panic(sdbErrorf("%w", err))
	}

	// TODO: add sdb tracing logger?
	return
}

// SetNonce sets the nonce of account.
// The nonce is a counter of the number of transactions sent from an account.
func (s *SDB) SetNonce(addr gethcommon.Address, nonce uint64) {
	acc := s.keeper.GetAccount(s.evmTxCtx, addr)
	if acc == nil {
		return
	}
	acc.Nonce = nonce
	err := s.keeper.SetAccount(s.evmTxCtx, addr, *acc)
	if err != nil {
		panic(sdbErrorf("%w", err))
	}
}

// SetCode sets the code of account.
// This function implements the [vm.StateDB] interface.
func (s *SDB) SetCode(addr gethcommon.Address, code []byte) {
	acc := s.keeper.GetAccount(s.evmTxCtx, addr)
	if acc == nil {
		acc = NewEmptyAccount() // Lazily create an empty account to attach code
	}

	codeHash := crypto.Keccak256Hash(code)
	codeHashBz := codeHash.Bytes()
	acc.CodeHash = codeHashBz

	// Persist account metadata (nonce, code hash)
	err := s.keeper.SetAccount(s.evmTxCtx, addr, *acc)
	if err != nil {
		panic(sdbErrorf("%w", err))
	}

	// TODO: Persist bytecode only if the code was not already set
	s.keeper.SetCode(s.evmTxCtx, codeHashBz, code)
}

// SetState sets the contract state.
func (s *SDB) SetState(
	addr gethcommon.Address, key, value gethcommon.Hash,
) (prevValue gethcommon.Hash) {
	prevValue = s.GetState(addr, key)
	var valueBz []byte
	if value == evm.EmptyHash {
		valueBz = nil
	} else {
		valueBz = value.Bytes()
	}
	s.keeper.SetState(s.evmTxCtx, addr, key, valueBz)
	return
}

func (s *SDB) subBalanceHoldingSupplyConstant(
	addr gethcommon.Address,
	wei *uint256.Int,
) {
	s.SubBalance(addr, wei, tracing.BalanceDecreaseSelfdestruct)
	s.AddBalance(evm.EVM_MODULE_ADDRESS, wei, tracing.BalanceIncreaseSelfdestruct)
}

func (s *SDB) hasSnapshotAccStatus(addr gethcommon.Address, change SnapshotAccChange) bool {
	gotChange, found := s.localState.AccountChangeMap[addr]
	for i := len(s.savedStates) - 1; !found && i >= 0; i-- {
		gotChange, found = s.localState.AccountChangeMap[addr]
	}
	if !found {
		return false
	}
	return gotChange == change
}

// SelfDestruct marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after [SelfDestruct].
func (s *SDB) SelfDestruct(addr gethcommon.Address) (prevWei uint256.Int) {
	s.localState.AccountChangeMap[addr] = SNAPSHOT_ACC_STATUS_DELETE
	prevWei = *s.GetBalance(addr)
	s.subBalanceHoldingSupplyConstant(addr, &prevWei)
	return prevWei

	// OLD IMPL

	// stateObject := s.getStateObject(addr)
	// if stateObject == nil {
	// 	return prevWei
	// }
	// prevWei = *(stateObject.Balance())
	// // Regardless of whether it is already destructed or not, we do have to
	// // journal the balance-change, if we set it to zero here.
	// if !stateObject.Balance().IsZero() {
	// 	stateObject.account.BalanceWei = new(uint256.Int)
	// }
	// // If it is already marked as self-destructed, we do not need to add it
	// // for journalling a second time.
	// if !stateObject.SelfDestructed {
	// 	s.Journal.append(suicideChange{
	// 		account:     &addr,
	// 		prev:        stateObject.SelfDestructed,
	// 		prevbalance: new(big.Int).Set(prevWei.ToBig()),
	// 	})
	// 	stateObject.SelfDestructed = true
	// }
	// return prevWei
}

// TODO: UD-DEBUG: SelfDestruct impl first
func (s *SDB) HasSelfDestructed(addr gethcommon.Address) bool {
	return s.hasSnapshotAccStatus(addr, SNAPSHOT_ACC_STATUS_DELETE)
	// s.localState
	// stateObject := s.getStateObject(addr)
	// if stateObject == nil {
	// 	return false
	// }
	// return stateObject.SelfDestructed
}

func (s *SDB) IsCreatedThisBlock(addr gethcommon.Address) bool {
	return s.hasSnapshotAccStatus(addr, SNAPSHOT_ACC_STATUS_CREATE)
}

// SelfDestruct6780 calls [SelfDesrtuct] only if the [stateObject] corresponding to
// the given "addr" was created this block.
//
// SelfDestruct6780 is post-EIP6780 selfdestruct, which means that it's a
// send-all-to-beneficiary, unless the contract was created in this same
// transaction, in which case it will be destructed.
// This method returns the prior balance, along with a boolean which is
// true iff the object was indeed destructed.
func (s *SDB) SelfDestruct6780(
	addr gethcommon.Address,
) (prevWei uint256.Int, isSelfDestructed bool) {
	if s.IsCreatedThisBlock(addr) {
		prevWei = s.SelfDestruct(addr)
		isSelfDestructed = true
		return
	}

	return *s.GetBalance(addr), s.hasSnapshotAccStatus(addr, SNAPSHOT_ACC_STATUS_DELETE)

	// stateObject := s.getStateObject(addr)
	// if stateObject == nil {
	// 	isSelfDestructed = false
	// } else if stateObject.createdThisBlock {
	// 	prevWei, isSelfDestructed = s.SelfDestruct(addr), true
	// } else {
	// 	prevWei, isSelfDestructed = *(stateObject.Balance()), false
	// }
	// return prevWei, isSelfDestructed
}

// Snapshot returns an identifier for the current revision of the state.
// TODO: UD-DEBUG: complete without tests
// FIXME: ? Are we caching in the right order.
func (s *SDB) Snapshot() int {
	branchedCtx := s.evmTxCtx.WithMultiStore(s.evmTxCtx.MultiStore().CacheMultiStore())
	s.savedCtxs = append(s.savedCtxs, s.evmTxCtx)
	s.evmTxCtx = branchedCtx
	s.savedStates = append(s.savedStates, s.localState)
	s.localState = NewLocalState()
	return s.SnapshotRevertIdx()
}

// SnapshotRevertIdx returns the current snapshot revert index. The original
// snapshot has revert index 0, and each subsequent snapshot afterward increments
// this value by 1.
func (s *SDB) SnapshotRevertIdx() int {
	return len(s.savedCtxs) - 1
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *SDB) RevertToSnapshot(revid int) {
	if currRevId := s.SnapshotRevertIdx(); revid > currRevId {
		// Only snapshot to valid reversion indices. Panic under same conditions
		// as Geth.
		panic(sdbErrorf("revision id %v cannot be reverted: current id is %d", revid, currRevId))
	}
	s.evmTxCtx = s.savedCtxs[revid]
	s.savedCtxs = s.savedCtxs[:revid]

	s.localState = s.savedStates[revid]
	s.savedStates = s.savedStates[:revid]

	s.Snapshot()
}

// sdbErrorf: wrapper of "fmt.Errorf" specific to the current Go package.
func sdbErrorf(format string, args ...any) error {
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
func (s *SDB) Commit() {
	// Empty self-destructed accounts
	{
		localStates := append(s.savedStates, s.localState)
		for i := len(localStates) - 1; i >= 0; i-- {
			localState := localStates[i]
			for addr, accChange := range localState.AccountChangeMap {
				if accChange != SNAPSHOT_ACC_STATUS_DELETE {
					continue
				}
				// Handle funds for the self-destructed account
				// TODO: Why send to the module? Why not?
				s.subBalanceHoldingSupplyConstant(addr, s.GetBalance(addr))
				// Delete self-destructed account from global state
				addrBech32 := eth.EthAddrToNibiruAddr(addr)
				acct := s.keeper.accountKeeper.GetAccount(s.evmTxCtx, addrBech32)
				if acct != nil {
					s.keeper.accountKeeper.RemoveAccount(s.evmTxCtx, acct)
				}
			}
		}
	}

	// Finalize all persistent state except `savedCtxs[0]` since it's the
	// original ctx to be committed by the baseapp.
	{
		ctxs := append(s.savedCtxs, s.evmTxCtx)
		for i := len(ctxs) - 1; i > 0; i-- {
			ctx := ctxs[i]
			ctx.MultiStore().(sdk.CacheMultiStore).Write()
		}
	}
}

// RootCtx returns the root context captured when the SDB was constructed.
// It is the base (anchor) context, and subsequent snapshots branch from it.
// Only the root context is ultimately committed by the baseapp.
//
// Only in [SDB.Commit] does the [SDB] write changes from all of the branched
// contexts, ultimately updating the root ctx to have the changes made to the
// current branched ctx ([SDB.Ctx]).
func (s *SDB) RootCtx() sdk.Context {
	return s.savedCtxs[0]
}

// TODO: cleanup - REMOVE
// CommitCacheCtx is identical to [SDB.Commit], except it:
// (1) uses the cacheCtx of the [SDB] and
// (2) does not save mutations of the cacheCtx to the commit context (s.evmTxCtx).
// The reason for (2) is that the overall EVM transaction (block, not internal)
// is only finalized when [Commit] is called, not when [CommitCacheCtx] is
// called.

// TODO: cleanup - REMOVE
// // SavePrecompileCalledJournalChange adds a snapshot of the commit multistore
// // ([PrecompileCalled]) to the [SDB] journal at the end of
// // successful invocation of a precompiled contract. This is necessary to revert
// // intermediate states where an EVM contract augments the multistore with a
// // precompile and an inconsistency occurs between the EVM module and other
// // modules.
// //
// // See [PrecompileCalled] for more info.
// func (s *SDB) SavePrecompileCalledJournalChange(
// 	journalChange PrecompileCalled,
// ) error {
// }

// Witness returns nil.
//
// Rationale: In Geth v1.14+, a [stateless.Witness] encompasses the state
// required to apply a set of transactions and derive a post state/receipt root.
//
// On Ethereum, this can be used to record trie (Merkle Patricia) accesses
// (storage and account), generate proofs for stateless clients, snap sync, or
// zkEVMs, and later reconstruct state from those proofs.
// In  other words, [Witness] is part of an effort toward stateless execution.
//
// NOTE: Nibiru does not use a Merkle Patricia Trie.
// Instead it uses IAVL over KVStore. That means there's no notion of
// Ethereum-style witnesses unless we simulate that separately.
//
// Thus, this function is optional to implement unless we build:
//   - zkEVM compatibility
//   - Stateless Ethereum clients
//   - Custom light client proofs that require a witness
func (s *SDB) Witness() *stateless.Witness {
	return nil
}

// ↓ If you remove the quotes below, golangci-lint will change the function name
// to American spelling as "Finalize", breaking interface compatibility.

// "Finalise"  prepares state objects at the end of a transaction execution.
//
// In Ethereum/Geth, this typically moves dirty storage to a pending layer,
// flushes prefetchers, and finalizes flags like newContract.
//
// Behavior: This matches Ethereum behavior (e.g., EIP-161 and EIP-6780 compatibility).
//   - If the account is non-empty, it clears the `newContract` flag.
//   - If the account is empty and deleteEmptyObjects is true, it removes it from live state.
//
// In Nibiru, [SDB.Finalise] can be a a no-op because:
//   - The Cosmos SDK state machine executes each transaction atomically.
//   - All writes happen against a cached multistore (`s.cacheCtx`) that gets committed
//     during `StateDB.Commit`.
//
// This function implements the [vm.StateDB] interface.
func (s *SDB) Finalise(deleteEmptyObjects bool) {
	// Empty self-destructed accounts
	{
		localStates := append(s.savedStates, s.localState)
		for i := len(localStates) - 1; i >= 0; i-- {
			localState := localStates[i]
			for addr, accChange := range localState.AccountChangeMap {
				if accChange != SNAPSHOT_ACC_STATUS_DELETE {
					continue
				}
				// TODO: Why send to the module? Why not?
				s.subBalanceHoldingSupplyConstant(addr, s.GetBalance(addr))
			}
		}
	}

	// Finalize all persistent state except `savedCtxs[0]` since it's the
	// original ctx to be committed by the baseapp.
	{
		ctxs := append(s.savedCtxs, s.evmTxCtx)
		for i := len(ctxs) - 1; i > 0; i-- {
			ctx := ctxs[i]
			ctx.MultiStore().(sdk.CacheMultiStore).Write()
		}
	}
}

// GetStorageRoot returns an empty state hash. This is done because a storage
// root make sense to implement for Nibiru, as it does not use Merkle Patricia
// Tries.
// This function implements the [vm.StateDB] interface.
func (s *SDB) GetStorageRoot(addr gethcommon.Address) (root gethcommon.Hash) {
	return root // or panic("unsupported")
}

// PointCache returns the point cache used by verkle tree.
// This function implements the [vm.StateDB] interface.
func (s *SDB) PointCache() *utils.PointCache {
	return nil
}

// --------------------------------------------------------
// LocalState
// --------------------------------------------------------

type LocalState struct {
	logs []*gethcore.Log

	AccountChangeMap map[gethcommon.Address]SnapshotAccChange
	ContractStorage  transientStorage

	// Gas refund counter for the state transition. Encoded as `uint64`. It is
	// valid for this field to be empty.
	gasRefund  []byte // gasRefund uint64
	accessList []byte // accessList json []gethcommon.Address

	SurplusWei *big.Int
}

func NewLocalState() *LocalState {
	return &LocalState{
		logs:             []*gethcore.Log{},
		AccountChangeMap: make(map[gethcommon.Address]SnapshotAccChange),
		ContractStorage:  make(transientStorage),
		gasRefund:        nil,
		accessList:       nil,
		SurplusWei:       big.NewInt(0),
	}
}

// SnapshotAccChange tracks changes in an account. Changes include:
// - an account marked for deletion (suicided).
// - an account created during the current EVM tx.
type SnapshotAccChange byte

var (
	SNAPSHOT_ACC_STATUS_CREATE SnapshotAccChange = 0x01
	SNAPSHOT_ACC_STATUS_DELETE SnapshotAccChange = 0x02
)

// transientStorage is a representation of EIP-1153 "Transient Storage".
type transientStorage map[gethcommon.Address]StorageForOneContract

// Set sets the transient-storage `value` for `key` at the given `addr`.
func (t transientStorage) Set(addr gethcommon.Address, key, value gethcommon.Hash) {
	if _, ok := t[addr]; !ok {
		t[addr] = make(StorageForOneContract)
	}
	t[addr][key] = value
}

// Get gets the transient storage for `key` at the given `addr`.
func (t transientStorage) Get(addr gethcommon.Address, key gethcommon.Hash) gethcommon.Hash {
	val, ok := t[addr]
	if !ok {
		return gethcommon.Hash{}
	}
	return val[key]
}

// Copy does a deep copy of the transientStorage
func (t transientStorage) Copy() transientStorage {
	storage := make(transientStorage)
	for key, value := range t {
		storage[key] = value.Copy()
	}
	return storage
}

// CopyForContract returns a deep copy of one [StorageForOneContract] map for a
// given smart contract. Returns an empty map if contract does not have transient
// storage.
func (t transientStorage) CopyForContract(
	addr gethcommon.Address,
) StorageForOneContract {
	if contStore, ok := t[addr]; ok {
		return contStore.Copy()
	}
	return make(StorageForOneContract)
}

// GetTransientState gets transient storage ([gethcommon.Hash]) for a given account.
func (s *SDB) GetTransientState(
	addr gethcommon.Address,
	key gethcommon.Hash,
) (stateVal gethcommon.Hash) {
	stateVal = s.localState.ContractStorage.Get(addr, key)
	if stateVal != evm.EmptyHash {
		return stateVal
	}
	for i := len(s.savedStates) - 1; i >= 0; i-- {
		stateVal = s.savedStates[i].ContractStorage.Get(addr, key)
		if stateVal != evm.EmptyHash {
			return stateVal
		}
	}
	return stateVal
}

func (s *SDB) GetTransientStorageForOneContract(
	addr gethcommon.Address,
) StorageForOneContract {
	stor := make(StorageForOneContract)
	states := append(s.savedStates, s.localState)
	for _, localState := range states {
		if localState == nil {
			continue
		}
		maps.Copy(stor, localState.ContractStorage.CopyForContract(addr))
	}
	return stor
}

func (s *SDB) GetStorageForOneContract(
	addr gethcommon.Address,
) StorageForOneContract {
	stor := make(StorageForOneContract)
	s.keeper.ForEachStorage(
		s.Ctx(), addr,
		func(key, value gethcommon.Hash) (keepGoing bool) {
			stor[key] = value
			return true
		})
	return stor
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (s *SDB) SetTransientState(
	addr gethcommon.Address,
	key, value gethcommon.Hash,
) {
	s.localState.ContractStorage.Set(addr, key, value)
}
