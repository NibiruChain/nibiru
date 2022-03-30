package common

import (
	"fmt"
	"sort"
)

const (
	GovDenom       = "umtrx"
	CollDenom      = "uust"
	GovPricePool   = "umtrx:uust"
	CollStablePool = "uusdm:uust"
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
