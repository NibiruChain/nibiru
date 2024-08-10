package omap

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func stringIsLess(a, b string) bool {
	return a < b
}

// ---------------------------------------------------------------------------
// SortedMap[string, V]
// ---------------------------------------------------------------------------

func SortedMap_String[V any](data map[string]V) SortedMap[string, V] {
	omap := SortedMap[string, V]{}
	return *omap.BuildFrom(data, stringSorter{})
}

// stringSorter is a Sorter implementation for keys of type string . It uses
// the built-in string comparison to determine order.
type stringSorter struct{}

var _ Sorter[string] = (*stringSorter)(nil)

func (sorter stringSorter) Less(a, b string) bool {
	return stringIsLess(a, b)
}

// ---------------------------------------------------------------------------
// SortedMap[asset.Pair, V]
// ---------------------------------------------------------------------------

func SortedMap_Pair[V any](
	data map[asset.Pair]V,
) SortedMap[asset.Pair, V] {
	omap := SortedMap[asset.Pair, V]{}
	return *omap.BuildFrom(data, pairSorter{})
}

// pairSorter is a Sorter implementation for keys of type asset.Pair. It uses
// the built-in string comparison to determine order.
type pairSorter struct{}

var _ Sorter[asset.Pair] = (*pairSorter)(nil)

func (sorter pairSorter) Less(a, b asset.Pair) bool {
	return stringIsLess(a.String(), b.String())
}

// ---------------------------------------------------------------------------
// SortedMap[gethcommon.Address, V]
// ---------------------------------------------------------------------------

func SortedMap_EthAddress[V any](
	data map[gethcommon.Address]V,
) SortedMap[gethcommon.Address, V] {
	return *new(SortedMap[gethcommon.Address, V]).BuildFrom(
		data, addrSorter{},
	)
}

// addrSorter implements "omap.Sorter" for the "gethcommon.Address" type.
type addrSorter struct{}

var _ Sorter[gethcommon.Address] = (*addrSorter)(nil)

func (s addrSorter) Less(a, b gethcommon.Address) bool {
	aInt := new(big.Int).SetBytes(a.Bytes())
	bInt := new(big.Int).SetBytes(b.Bytes())
	return aInt.Cmp(bInt) < 0
}
