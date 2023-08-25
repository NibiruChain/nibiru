package omap_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/omap"
)

// TestLenHasKeys checks the length of the ordered map and verifies if the map
// contains certain keys.
func TestLenHasKeys(t *testing.T) {
	type HasCheck struct {
		key string
		has bool
	}

	testCases := []struct {
		dataMap   map[string]int
		len       int
		hasChecks []HasCheck
	}{
		{
			dataMap: map[string]int{"xyz": 420, "abc": 69},
			len:     2,
			hasChecks: []HasCheck{
				{key: "foo", has: false},
				{key: "xyz", has: true},
				{key: "bar", has: false},
			},
		},
		{
			dataMap: map[string]int{"aaa": 420, "bbb": 69, "ccc": 69, "ddd": 28980},
			len:     4,
			hasChecks: []HasCheck{
				{key: "foo", has: false},
				{key: "xyz", has: false},
				{key: "bbb", has: true},
			},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("case-%d", idx), func(t *testing.T) {
			om := omap.OrderedMap_String[int](tc.dataMap)

			require.Equal(t, tc.len, om.Len())

			orderedKeys := om.Keys()
			definitelyOrderedKeys := []string{}
			definitelyOrderedKeys = append(definitelyOrderedKeys, orderedKeys...)
			sort.Strings(definitelyOrderedKeys)

			require.Equal(t, definitelyOrderedKeys, orderedKeys)

			idx := 0
			for key := range om.Range() {
				require.Equal(t, orderedKeys[idx], key)
				idx++
			}
		})
	}
}

// TestGetSetDelete checks the Get, Set, and Delete operations on the OrderedMap.
func TestGetSetDelete(t *testing.T) {
	om := omap.OrderedMap_String[string](make(map[string]string))
	require.Equal(t, 0, om.Len())

	om.Set("foo", "fooval")
	require.True(t, om.Has("foo"))
	require.Equal(t, 1, om.Len())

	om.Delete("bar") // shouldn't cause any problems
	om.Delete("foo")
	require.False(t, om.Has("foo"))
	require.Equal(t, 0, om.Len())
}

// TestOrderedMap_Pair tests an OrderedMap where the key is an asset.Pair, a
// type that isn't built-in.
func TestOrderedMap_Pair(t *testing.T) {
	pairStrs := []string{
		"abc:xyz", "abc:abc", "aaa:bbb", "xyz:xyz", "bbb:ccc", "xyz:abc",
	}
	orderedKeyStrs := []string{}
	orderedKeyStrs = append(orderedKeyStrs, pairStrs...)
	sort.Strings(orderedKeyStrs)

	orderedKeys := asset.MustNewPairs(orderedKeyStrs...)
	pairs := asset.MustNewPairs(pairStrs...)

	type ValueType struct{}
	unorderedMap := make(map[asset.Pair]ValueType)
	for _, pair := range pairs {
		unorderedMap[pair] = ValueType{}
	}

	om := omap.OrderedMap_Pair[ValueType](unorderedMap)
	require.Equal(t, 6, om.Len())
	require.EqualValues(t, orderedKeys, om.Keys())
	require.NotEqualValues(t, asset.PairsToStrings(orderedKeys), pairStrs)

	var pairsFromLoop []asset.Pair
	for pair := range om.Range() {
		pairsFromLoop = append(pairsFromLoop, pair)
	}
	require.EqualValues(t, orderedKeys, pairsFromLoop)
	require.NotEqualValues(t, pairsFromLoop, pairs)
}
