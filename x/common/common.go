package common

import (
	"fmt"
	"sort"
)

var (
	GovDenom    = "umtrx"
	CollDenom   = "uust"
	StableDenom = "uusdm"

	TreasuryPoolModuleAccount = "treasury_pool"

	WhitelistedColl = []string{CollDenom}

	GovCollPool    = PoolNameFromDenoms([]string{GovDenom, CollDenom})
	GovStablePool  = PoolNameFromDenoms([]string{GovDenom, StableDenom})
	CollStablePool = PoolNameFromDenoms([]string{CollDenom, StableDenom})
)

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
