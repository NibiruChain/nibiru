package common

import (
	"fmt"
	"sort"
	"strings"
)

var (
	GovDenom        = "unibi"
	CollDenom       = "uust"
	StableDenom     = "unusd"
	StakeTokenDenom = "stake"

	TreasuryPoolModuleAccount = "treasury_pool"

	PairSeparator = ":"

	WhitelistedColl = []string{CollDenom}

	GovStablePool  = AssetPair{Token0: GovDenom, Token1: StableDenom}
	CollStablePool = AssetPair{Token0: CollDenom, Token1: StableDenom}

	ErrInvalidTokenPair = fmt.Errorf("invalid token pair")
)

func NewAssetPairFromStr(pair string) (AssetPair, error) {
	split := strings.Split(pair, PairSeparator)
	if len(split) != 2 {
		return AssetPair{}, ErrInvalidTokenPair
	}

	if split[0] == "" || split[1] == "" {
		return AssetPair{}, ErrInvalidTokenPair
	}

	return AssetPair{Token0: split[0], Token1: split[1]}, nil
}

type AssetPair struct {
	Token0 string
	Token1 string
}

// Name is the name of the pool that corresponds to the two assets on this pair.
func (pair AssetPair) Name() string {
	return PoolNameFromDenoms([]string{pair.Token0, pair.Token1})
}

func (pair AssetPair) PairID() string {
	return pair.Name()
}

func (pair AssetPair) String() string {
	return fmt.Sprintf("%s%s%s", pair.Token0, PairSeparator, pair.Token1)
}

func (pair AssetPair) IsProperOrder() bool {
	return pair.Name() == pair.String()
}

func (pair AssetPair) Inverse() AssetPair {
	return AssetPair{pair.Token1, pair.Token0}
}

func (pair AssetPair) Proper() AssetPair {
	if pair.IsProperOrder() {
		return pair
	} else {
		return pair.Inverse()
	}
}

func (pair AssetPair) GetBaseTokenDenom() string {
	return pair.Token0
}

func (pair AssetPair) GetQuoteTokenDenom() string {
	return pair.Token1
}

func DenomsFromPoolName(pool string) (denoms []string) {
	return strings.Split(pool, ":")
}

// PoolNameFromDenoms returns a sorted string representing a pool of assets
func PoolNameFromDenoms(denoms []string) string {
	sort.Strings(denoms) // alphabetically sort in-place
	return RawPoolNameFromDenoms(denoms)
}

// RawPoolNameFromDenoms returns a string representing a pool of assets in the
// exact order the denoms were given as args
func RawPoolNameFromDenoms(denoms []string) string {
	poolName := denoms[0]
	for idx, denom := range denoms {
		if idx != 0 {
			poolName += fmt.Sprintf("%s%s", PairSeparator, denom)
		}
	}
	return poolName
}
