package statedb

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var emptyCodeHash = crypto.Keccak256(nil)

// Account represents an Ethereum account as viewed by the Auth module state. The
// balance is stored in the smallest native unit (e.g., micronibi or unibi).
// These objects are stored in the storage of auth module.
type Account struct {
	// BalanceNative is the micronibi (unibi) balance of the account, which is
	// the official balance in the x/bank module state
	BalanceNative *uint256.Int
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
	BalanceWei *uint256.Int
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
		BalanceWei: evm.NativeToWeiU256(acc.BalanceNative),
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
		BalanceNative: evm.WeiToNativeMustU256(acc.BalanceWei.ToBig()),
		Nonce:         acc.Nonce,
		CodeHash:      acc.CodeHash,
	}
}

// NewEmptyAccount returns an empty account.
func NewEmptyAccount() *Account {
	return &Account{
		BalanceNative: new(uint256.Int),
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

func (s Storage) Copy() Storage {
	cpy := make(Storage, len(s))
	for key, value := range s {
		cpy[key] = value
	}
	return cpy
}

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}
	return
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
	OriginStorage Storage
	DirtyStorage  Storage

	address common.Address

	// flags
	DirtyCode        bool
	SelfDestructed   bool // True if the stateObject has been self destructed
	createdThisBlock bool // True if the stateObject was created this block
	// This is an EIP-6780 flag indicating whether the object is eligible for
	// self-destruct according to EIP-6780. The flag could be set either when
	// the contract is just created within the current transaction, or when the
	// object was previously existent and is being deployed as a contract within
	// the current transaction.
	newContract bool
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, account *Account) *stateObject {
	createdThisBlock := account == nil
	if createdThisBlock {
		account = &Account{}
	}
	if account.BalanceNative == nil {
		account.BalanceNative = new(uint256.Int)
	}
	if account.CodeHash == nil {
		account.CodeHash = emptyCodeHash
	}
	return &stateObject{
		db:      db,
		address: address,
		// Reflect the micronibi (unibi) balance in wei
		account:          account.ToWei(),
		OriginStorage:    make(Storage),
		DirtyStorage:     make(Storage),
		createdThisBlock: createdThisBlock,
	}
}

// isEmpty returns whether the account is considered isEmpty.
func (s *stateObject) isEmpty() bool {
	return s.account.Nonce == 0 &&
		s.account.BalanceWei.Sign() == 0 &&
		bytes.Equal(s.account.CodeHash, emptyCodeHash)
}

func (s *stateObject) setBalance(amount *big.Int) {
	s.account.BalanceWei = uint256.MustFromBig(amount)
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
	code := s.db.keeper.GetCode(s.db.evmTxCtx, common.BytesToHash(s.CodeHash()))
	s.code = code
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none.
func (s *stateObject) CodeSize() int {
	return len(s.Code())
}

// CodeHash returns the code hash of account
func (s *stateObject) CodeHash() []byte {
	return s.account.CodeHash
}

// Balance returns the balance of account
func (s *stateObject) Balance() *uint256.Int {
	return s.account.BalanceWei
}

// Copied from /core/state/journal.go in geth v1.14
var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

// Copied from /core/state/state_object.go in geth v1.14
func (s *stateObject) touch() {
	addr := s.address
	s.db.Journal.append(touchChange{
		account: addr,
	})
	if addr == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated
		// from flattened journals.
		s.db.Journal.dirty(addr)
	}
}
