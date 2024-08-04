// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"
	"math/big"

	"github.com/NibiruChain/collections"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName string name of module
	ModuleName = "evm"

	// StoreKey: Persistent storage key for ethereum storage data, account code
	// (StateDB) or block related data for the Eth Web3 API.
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

	// KV store prefix for `FunToken` mappings
	KeyPrefixFunTokens
	// KV store prefix for indexing `FunToken` by ERC-20 address
	KeyPrefixFunTokenIdxErc20
	// KV store prefix for indexing `FunToken` by bank coin denomination
	KeyPrefixFunTokenIdxBankDenom
)

// KVStore transient prefix namespaces for the EVM Module. Transient stores only
// remain for current block, and have more gas efficient read and write access.
const (
	NamespaceBlockBloom collections.Namespace = iota + 1
	NamespaceBlockTxIndex
	NamespaceBlockLogSize
	NamespaceBlockGasUsed
)

var KeyPrefixBzAccState = KeyPrefixAccState.Prefix()

// PrefixAccStateEthAddr returns a prefix to iterate over a given account storage.
func PrefixAccStateEthAddr(address gethcommon.Address) []byte {
	return append(KeyPrefixBzAccState, address.Bytes()...)
}

// StateKey defines the full key under which an account state is stored.
func StateKey(address gethcommon.Address, key []byte) []byte {
	return append(PrefixAccStateEthAddr(address), key...)
}

const (
	// Amino names
	updateParamsName = "evm/MsgUpdateParams"
)

type CallType int

const (
	// CallTypeRPC call type is used on requests to eth_estimateGas rpc API endpoint
	CallTypeRPC CallType = iota + 1
	// CallTypeSmart call type is used in case of smart contract methods calls
	CallTypeSmart
)

// ModuleAddressEVM: Module account address as a `gethcommon.Address`.
func ModuleAddressEVM() gethcommon.Address {
	if evmModuleAddr == zeroAddr {
		evmModuleAddr = gethcommon.BytesToAddress(
			authtypes.NewModuleAddress(ModuleName).Bytes(),
		)
	}
	return evmModuleAddr
}

var (
	zeroAddr      gethcommon.Address
	evmModuleAddr gethcommon.Address
)

// NativeToWei converts a "unibi" amount to "wei" units for the EVM.
//
// Micronibi, labeled "unibi", is the base denomination for NIBI. For NIBI to be
// considered "ether" by Ethereum clients, we need to follow the constraint
// equation: 1 NIBI = 10^18 wei.
//
// Since 1 NIBI = 10^6 micronibi = 10^6 unibi, the following is true:
// 10^0 unibi == 10^12 wei
func NativeToWei(evmDenomAmount *big.Int) (weiAmount *big.Int) {
	pow10 := new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
	return new(big.Int).Mul(evmDenomAmount, pow10)
}

// WeiToNative converts a "wei" amount to "unibi" units.
//
// Micronibi, labeled "unibi", is the base denomination for NIBI. For NIBI to be
// considered "ether" by Ethereum clients, we need to follow the constraint
// equation: 1 NIBI = 10^18 wei.
//
// Since 1 NIBI = 10^6 micronibi = 10^6 unibi, the following is true:
// 10^0 unibi == 10^12 wei
func WeiToNative(weiAmount *big.Int) (evmDenomAmount *big.Int) {
	pow10 := new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
	return new(big.Int).Quo(weiAmount, pow10)
}

// ParseWeiAsMultipleOfMicronibi truncates the given wei amount to the highest
// multiple of 1 micronibi (10^12 wei). It returns the truncated value and an
// error if the input value is too small.
//
// Args:
//   - weiInt (*big.Int): The amount of wei to be parsed.
//
// Returns:
//   - newWeiInt (*big.Int): The truncated amount of wei, which is a multiple of 1 micronibi.
//   - err (error): An error indicating if the input value is within the range
//     (1, 10^12) inclusive.
//
// Example:
//
//	Input  number:  123456789012345678901234567890
//	Parsed number:  123456789012 * 10^12
func ParseWeiAsMultipleOfMicronibi(weiInt *big.Int) (newWeiInt *big.Int, err error) {
	// if "weiValue" is nil, 0, or negative, early return
	if weiInt == nil || !(weiInt.Cmp(big.NewInt(0)) > 0) {
		return weiInt, nil
	}

	// err if weiInt is too small
	tenPow12 := new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
	if weiInt.Cmp(tenPow12) < 0 {
		return weiInt, fmt.Errorf(
			"wei amount is too small (%s), cannot transfer less than 1 micronibi. 10^18 wei == 1 NIBI == 10^6 micronibi", weiInt)
	}

	// truncate to highest micronibi amount
	newWeiInt = NativeToWei(WeiToNative(weiInt))
	return newWeiInt, nil
}
