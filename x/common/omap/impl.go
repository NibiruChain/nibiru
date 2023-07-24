package omap

import (
	"github.com/NibiruChain/nibiru/x/common/asset"
)

func stringIsLess(a, b string) bool {
	return a < b
}

// ---------------------------------------------------------------------------
// OrderedMap[string, V]: OrderedMap_String
// ---------------------------------------------------------------------------

func OrderedMap_String[V any](data map[string]V) OrderedMap[string, V] {
	omap := OrderedMap[string, V]{}
	return omap.BuildFrom(data, stringSorter{})
}

// stringSorter is a Sorter implementation for keys of type string . It uses
// the built-in string comparison to determine order.
type stringSorter struct{}

var _ Sorter[string] = (*stringSorter)(nil)

func (sorter stringSorter) Less(a, b string) bool {
	return stringIsLess(a, b)
}

// ---------------------------------------------------------------------------
// OrderedMap[asset.Pair, V]: OrderedMap_Pair
// ---------------------------------------------------------------------------

func OrderedMap_Pair[V any](
	data map[asset.Pair]V,
) OrderedMap[asset.Pair, V] {
	omap := OrderedMap[asset.Pair, V]{}
	return omap.BuildFrom(data, pairSorter{})
}

// pairSorter is a Sorter implementation for keys of type asset.Pair. It uses
// the built-in string comparison to determine order.
type pairSorter struct{}

var _ Sorter[asset.Pair] = (*pairSorter)(nil)

func (sorter pairSorter) Less(a, b asset.Pair) bool {
	return stringIsLess(a.String(), b.String())
}
