package types

import (
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "dex"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_dex"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

var (
	// KeyNextGlobalPoolNumber defines key to store the next Pool ID to be used
	KeyNextGlobalPoolNumber = []byte{0x01}
	// KeyPrefixPools defines prefix to store pools
	KeyPrefixPools = []byte{0x02}
	// KeyTotalLiquidity defines key to store total liquidity
	KeyTotalLiquidity = []byte{0x03}
	// KeyPrefixPoolIds defines prefix to store pool ids by denoms in the pool
	KeyPrefixPoolIds = []byte{0x04}
)

func GetDenomPrefixPoolIds(denoms ...string) []byte {
	sort.Strings(denoms)
	concatenation := strings.Join(denoms[:], "")
	return append(KeyPrefixPoolIds, []byte(concatenation)...)
}

func GetKeyPrefixPools(poolId uint64) []byte {
	return append(KeyPrefixPools, sdk.Uint64ToBigEndian(poolId)...)
}

func GetDenomLiquidityPrefix(denom string) []byte {
	return append(KeyTotalLiquidity, []byte(denom)...)
}
