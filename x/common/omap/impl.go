package omap

import (
	"math/big"

	"github.com/NibiruChain/nibiru/x/common/asset"
	gethcommon "github.com/ethereum/go-ethereum/common"
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

func OrderedMap_EthAddress[V any](
	data map[gethcommon.Address]V,
) OrderedMap[gethcommon.Address, V] {
	return OrderedMap[gethcommon.Address, V]{}.BuildFrom(
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
