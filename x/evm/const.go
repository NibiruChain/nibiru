// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"github.com/NibiruChain/collections"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName string name of module
	ModuleName = "evm"

	// StoreKey: Persistent storage key for ethereum storage data, account code (StateDB) or block
	// related data for the Eth Web3 API.
	StoreKey = ModuleName

	// TransientKey is the key to access the EVM transient store, that is reset
	// during the Commit phase.
	TransientKey = "transient_" + ModuleName

	// RouterKey uses module name for routing
	RouterKey = ModuleName
)

// prefix bytes for the EVM persistent store
const (
	KeyPrefixAccCodes collections.Namespace = iota + 1
	KeyPrefixAccState
	KeyPrefixParams
	KeyPrefixEthAddrIndex
)

// prefix bytes for the EVM transient store
const (
	prefixTransientBloom collections.Namespace = iota + 1
	prefixTransientTxIndex
	prefixTransientLogSize
	prefixTransientGasUsed
)

// KVStore key prefixes
var (
	KeyPrefixBzAccState = KeyPrefixAccState.Prefix()
)

// Transient Store key prefixes
var (
	KeyPrefixTransientBloom   = prefixTransientBloom.Prefix()
	KeyPrefixTransientTxIndex = prefixTransientTxIndex.Prefix()
	KeyPrefixTransientLogSize = prefixTransientLogSize.Prefix()
	KeyPrefixTransientGasUsed = prefixTransientGasUsed.Prefix()
)

// PrefixAccStateEthAddr returns a prefix to iterate over a given account storage.
func PrefixAccStateEthAddr(address common.Address) []byte {
	return append(KeyPrefixBzAccState, address.Bytes()...)
}

// StateKey defines the full key under which an account state is stored.
func StateKey(address common.Address, key []byte) []byte {
	return append(PrefixAccStateEthAddr(address), key...)
}
