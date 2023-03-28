package types

import (
	"fmt"
	"strings"
)

// String returns the string representation of the pool. Note that this differs
// from the default output of the proto-generated 'String' method.
func (pool *Vpool) String() string {
	elems := []string{
		fmt.Sprintf("pair: %s", pool.Pair),
		fmt.Sprintf("base_reserves: %s", pool.BaseAssetReserve),
		fmt.Sprintf("quote_reserves: %s", pool.QuoteAssetReserve),
		fmt.Sprintf("sqrt_depth: %s", pool.SqrtDepth),
		fmt.Sprintf("config: %s", &pool.Config),
	}
	elemString := strings.Join(elems, ", ")
	return "{ " + elemString + " }"
}
