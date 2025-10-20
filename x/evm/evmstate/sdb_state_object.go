package evmstate

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"bytes"
	"fmt"
	"sort"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/evm"
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
