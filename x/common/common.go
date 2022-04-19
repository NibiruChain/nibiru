package common

import (
	"fmt"
	"sort"
)

var (
	GovDenom    = "unibi"
	CollDenom   = "uust"
	StableDenom = "unusd"

	TreasuryPoolModuleAccount = "treasury_pool"

	WhitelistedColl = []string{CollDenom}

	GovCollPool    = Pair{GovDenom, CollDenom}
	GovStablePool  = Pair{GovDenom, StableDenom}
	CollStablePool = Pair{CollDenom, StableDenom}
)

type Pair struct {
	Token0 string
	Token1 string
}

// name is the name of the pool that corresponds to the two assets on this pair.
func (pair Pair) Name() string {
	return PairNameFromDenoms([]string{pair.Token0, pair.Token1})
}

func (pair Pair) String() string {
	return fmt.Sprintf("%s:%s", pair.Token0, pair.Token1)
}

func (pair Pair) IsProperOrder() bool {
	return pair.Name() == pair.String()
}

func (pair Pair) Inverse() Pair {
	return Pair{pair.Token1, pair.Token0}
}

func PairNameFromDenoms(denoms []string) string {
	sort.Strings(denoms) // alphabetically sort in-place
	poolName := denoms[0]
	for idx, denom := range denoms {
		if idx != 0 {
			poolName += fmt.Sprintf(":%s", denom)
		}
	}
	return poolName
}
