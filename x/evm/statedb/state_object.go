// Copyright (c) 2023-2024 Nibi, Inc.
package statedb

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var emptyCodeHash = crypto.Keccak256(nil)

// Account represents an Ethereum account as viewed by the Auth module state. The
// balance is stored in the smallest native unit (e.g., micronibi or unibi).
// These objects are stored in the storage of auth module.
type Account struct {
	// BalanceNative is the micronibi (unibi) balance of the account, which is
	// the official balance in the x/bank module state
	BalanceNative *big.Int
	// Nonce is the number of transactions sent from this account or, for contract accounts, the number of contract-creations made by this account
	Nonce uint64
	// CodeHash is the hash of the contract code for this account, or nil if it's not a contract account
	CodeHash []byte
}

// AccountWei represents an Ethereum account as viewed by the EVM. This struct is
// derived from an `Account` but represents balances in wei, which is necessary
// for correct operation within the EVM. The EVM expects and operates on wei
// values, which are 10^12 times larger than the native unibi value due to the
// definition of NIBI as "ether".
type AccountWei struct {
	BalanceWei *big.Int
	// Nonce is the number of transactions sent from this account or, for contract accounts, the number of contract-creations made by this account
	Nonce uint64
	// CodeHash is the hash of the contract code for this account, or nil if it's not a contract account
	CodeHash []byte
}

// ToWei converts an Account (native representation) to AccountWei (EVM
// representation). This conversion is necessary when moving from the Cosmos SDK
// context to the EVM context. It multiplies the balance by 10^12 to convert from
// unibi to wei.
func (acc Account) ToWei() AccountWei {
	return AccountWei{
		BalanceWei: evm.NativeToWei(acc.BalanceNative),
		Nonce:      acc.Nonce,
		CodeHash:   acc.CodeHash,
	}
}

// ToNative converts an AccountWei (EVM representation) back to an Account
// (native representation). This conversion is necessary when moving from the EVM
// context back to the Cosmos SDK context. It divides the balance by 10^12 to
// convert from wei to unibi.
func (acc AccountWei) ToNative() Account {
	return Account{
		BalanceNative: evm.WeiToNative(acc.BalanceWei),
		Nonce:         acc.Nonce,
		CodeHash:      acc.CodeHash,
	}
}

// NewEmptyAccount returns an empty account.
func NewEmptyAccount() *Account {
	return &Account{
		BalanceNative: new(big.Int),
		CodeHash:      emptyCodeHash,
	}
}

// IsContract returns if the account contains contract code.
func (acct *Account) IsContract() bool {
	return (acct != nil) && !bytes.Equal(acct.CodeHash, emptyCodeHash)
}

// Storage represents in-memory cache/buffer of contract storage.
type Storage map[common.Hash]common.Hash

// SortedKeys sort the keys for deterministic iteration
func (s Storage) SortedKeys() []common.Hash {
	keys := make([]common.Hash, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) < 0
	})
	return keys
}

// stateObject represents the state of a Nibiru EVM account.
// It encapsulates both the account data (balance, nonce, code) and the contract
// storage state. stateObject serves as an in-memory cache and staging area for
// changes before they are committed to the underlying storage.
//
// Key features:
// 1. It uses AccountWei, which represents balances in wei for EVM compatibility.
// 2. It maintains both the original (committed) storage and dirty (uncommitted) storage.
// 3. It tracks whether the account has been marked for deletion (suicided).
// 4. It caches the contract code for efficient access.
//
// stateObjects are used to:
// - Efficiently manage and track changes to account state during EVM execution.
// - Provide a layer of abstraction between the EVM and the underlying storage.
// - Enable features like state reverting and snapshotting.
// - Optimize performance by minimizing direct access to the underlying storage.
type stateObject struct {
	db *StateDB

	account AccountWei
	code    []byte

	// state storage
	originStorage Storage
	dirtyStorage  Storage

	address common.Address

	// flags
	dirtyCode bool
	suicided  bool
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, account Account) *stateObject {
	if account.BalanceNative == nil {
		account.BalanceNative = new(big.Int)
	}
	if account.CodeHash == nil {
		account.CodeHash = emptyCodeHash
	}
	return &stateObject{
		db:      db,
		address: address,
		// Reflect the micronibi (unibi) balance in wei
		account:       account.ToWei(),
		originStorage: make(Storage),
		dirtyStorage:  make(Storage),
	}
}

// isEmpty returns whether the account is considered isEmpty.
func (s *stateObject) isEmpty() bool {
	return s.account.Nonce == 0 &&
		s.account.BalanceWei.Sign() == 0 &&
		bytes.Equal(s.account.CodeHash, emptyCodeHash)
}

// AddBalance adds amount to s's balance.
// It is used to add funds to the destination account of a transfer.
func (s *stateObject) AddBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Add(s.Balance(), amount))
}

// SubBalance removes amount from s's balance.
// It is used to remove funds from the origin account of a transfer.
func (s *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Sub(s.Balance(), amount))
}

// SetBalance update account balance.
func (s *stateObject) SetBalance(amount *big.Int) {
	s.db.journal.append(balanceChange{
		account: &s.address,
		prevWei: new(big.Int).Set(s.account.BalanceWei),
	})
	s.setBalance(amount)
}

func (s *stateObject) setBalance(amount *big.Int) {
	s.account.BalanceWei = amount
}

//
// Attribute accessors
//

// Address returns the address of the contract/account
func (s *stateObject) Address() common.Address {
	return s.address
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code() []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), emptyCodeHash) {
		return nil
	}
	code := s.db.keeper.GetCode(s.db.ctx, common.BytesToHash(s.CodeHash()))
	s.code = code
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none.
func (s *stateObject) CodeSize() int {
	return len(s.Code())
}

// SetCode set contract code to account
func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := s.Code()
	s.db.journal.append(codeChange{
		account:  &s.address,
		prevhash: s.CodeHash(),
		prevcode: prevcode,
	})
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.account.CodeHash = codeHash[:]
	s.dirtyCode = true
}

// SetNonce set nonce to account
func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: &s.address,
		prev:    s.account.Nonce,
	})
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.account.Nonce = nonce
}

// CodeHash returns the code hash of account
func (s *stateObject) CodeHash() []byte {
	return s.account.CodeHash
}

// Balance returns the balance of account
func (s *stateObject) Balance() *big.Int {
	return s.account.BalanceWei
}

// Nonce returns the nonce of account
func (s *stateObject) Nonce() uint64 {
	return s.account.Nonce
}

// GetCommittedState query the committed state
func (s *stateObject) GetCommittedState(key common.Hash) common.Hash {
	if value, cached := s.originStorage[key]; cached {
		return value
	}
	// If no live objects are available, load it from keeper
	value := s.db.keeper.GetState(s.db.ctx, s.Address(), key)
	s.originStorage[key] = value
	return value
}

// GetState query the current state (including dirty state)
func (s *stateObject) GetState(key common.Hash) common.Hash {
	if value, dirty := s.dirtyStorage[key]; dirty {
		return value
	}
	return s.GetCommittedState(key)
}

// SetState sets the contract state
func (s *stateObject) SetState(key common.Hash, value common.Hash) {
	// If the new value is the same as old, don't set
	prev := s.GetState(key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:  &s.address,
		key:      key,
		prevalue: prev,
	})
	s.setState(key, value)
}

func (s *stateObject) setState(key, value common.Hash) {
	s.dirtyStorage[key] = value
}
