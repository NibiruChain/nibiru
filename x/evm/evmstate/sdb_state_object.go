package evmstate

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// Account represents an Ethereum account according to the Auth and Bank module
// state.
type Account struct {
	// BalanceNwei is "NIBI wei", or attoNIBI, balance from the x/bank module
	// state. It has the same relationship with NIBI
	// that wei has with ETH on Ethereum.
	// Therefore, 10^{18} nwei := 1 NIBI. Equivalently, one micronibi (unibi) is
	// 10^{12} nwei.
	BalanceNwei *uint256.Int

	// Nonce is the number of transactions sent from this account or, for contract accounts, the number of contract-creations made by this account
	Nonce uint64
	// CodeHash is the hash of the contract code for this account, or nil if it's not a contract account
	CodeHash []byte
}

// NewEmptyAccount returns an empty account.
func NewEmptyAccount() *Account {
	return &Account{
		BalanceNwei: new(uint256.Int),
		CodeHash:    evm.EmptyCodeHashBz,
	}
}

// IsContract returns if the account contains contract code.
func (acct *Account) IsContract() bool {
	return (acct != nil) && !bytes.Equal(acct.CodeHash, evm.EmptyCodeHashBz)
}

// StorageForOneContract represents an in-memory cache of contract storage.
// In the EVM, contract storage is the mapping from slot key -> slot value, where
// both are 32-byte arrays of type [gethcommon.Hash].
//
// This concept comes from the Ethereum Yellow Paper §4.1: Each account
// maintains a mapping from 256-bit words to 256-bit words, called "storage".
type StorageForOneContract map[gethcommon.Hash]gethcommon.Hash

// SortedKeys sort the keys for deterministic iteration
func (s StorageForOneContract) SortedKeys() []gethcommon.Hash {
	keys := make([]gethcommon.Hash, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) < 0
	})
	return keys
}

func (s StorageForOneContract) Copy() StorageForOneContract {
	cpy := make(StorageForOneContract, len(s))
	for key, value := range s {
		cpy[key] = value
	}
	return cpy
}

func (s StorageForOneContract) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}
	return
}

// TODO: UD-DEBUG: What can be extracted from the stateObject docs and brought
// over.
//
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
// type stateObject struct {
// 	sdb *SDB
// 	account AccountWei
// 	code    []byte
// 	// state storage
// 	OriginStorage StorageForOneContract
// 	DirtyStorage  StorageForOneContract
// 	address gethcommon.Address
// }

// TODO: Read up on EIP-6780 and make sure it's honored.
//
// (deleted) is an EIP-6780 flag indicating whether the object is eligible for
// self-destruct according to EIP-6780. The flag could be set either when
// the contract is just created within the current transaction, or when the
// object was previously existent and is being deployed as a contract within
// the current transaction.

// Copied from /core/state/journal.go in geth v1.14
var ripemd = gethcommon.HexToAddress("0000000000000000000000000000000000000003")
