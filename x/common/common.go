package common

import (
	"fmt"
	"sort"
	"strings"
)

var (
	GovDenom    = "unibi"
	CollDenom   = "uust"
	StableDenom = "unusd"

	TreasuryPoolModuleAccount = "treasury_pool"

	WhitelistedColl = []string{CollDenom}

	GovCollPool    = AssetPair{GovDenom, CollDenom}
	GovStablePool  = AssetPair{GovDenom, StableDenom}
	CollStablePool = AssetPair{CollDenom, StableDenom}
)

type AssetPair struct {
	Token0 string
	Token1 string
}

// name is the name of the pool that corresponds to the two assets on this pair.
func (pair AssetPair) Name() string {
	return PoolNameFromDenoms([]string{pair.Token0, pair.Token1})
}

func (pair AssetPair) String() string {
	return fmt.Sprintf("%s:%s", pair.Token0, pair.Token1)
}

func (pair AssetPair) IsProperOrder() bool {
	return pair.Name() == pair.String()
}

func (pair AssetPair) Inverse() AssetPair {
	return AssetPair{pair.Token1, pair.Token0}
}

func NewAssetPair(token0 string, token1 string) AssetPair {
	return AssetPair{Token0: token0, Token1: token1}
}

func DenomsFromPoolName(pool string) (denoms []string) {
	return strings.Split(pool, ":")
}

// PoolNameFromDenoms returns a sorted string representing a pool of assets
func PoolNameFromDenoms(denoms []string) string {
	sort.Strings(denoms) // alphabetically sort in-place
	poolName := denoms[0]
	for idx, denom := range denoms {
		if idx != 0 {
			poolName += fmt.Sprintf(":%s", denom)
		}
	}
	return poolName
}

// RawPoolNameFromDenoms returns a string representing a pool of assets in the
// exact order the denoms were given as args
func RawPoolNameFromDenoms(denoms ...string) string {
	poolName := denoms[0]
	for idx, denom := range denoms {
		if idx != 0 {
			poolName += fmt.Sprintf(":%s", denom)
		}
	}
	return poolName
}
