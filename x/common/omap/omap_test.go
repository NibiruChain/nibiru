package omap_test

import (
	"fmt"
	"math/big"
	"sort"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/omap"
)

type Suite struct {
	suite.Suite
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(Suite))
}

// TestLenHasKeys checks the length of the sorted map and verifies if the map
// contains certain keys.
func (s *Suite) TestLenHasKeys() {
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
		s.Run(fmt.Sprintf("case-%d", idx), func() {
			om := omap.SortedMap_String(tc.dataMap)

			s.Require().Equal(tc.len, om.Len())

			sortedKeys := om.Keys()
			definitelySortedKeys := []string{}
			definitelySortedKeys = append(definitelySortedKeys, sortedKeys...)
			sort.Strings(definitelySortedKeys)

			s.Equal(definitelySortedKeys, sortedKeys)

			idx := 0
			for key := range om.Range() {
				s.Equal(sortedKeys[idx], key)
				idx++
			}
		})
	}
}

// TestGetSetDelete checks the Get, Set, and Delete operations on the SortedMap.
func (s *Suite) TestGetSetDelete() {
	s.Run("single element", func() {
		om := omap.SortedMap_String(make(map[string]string))
		s.Require().Equal(0, om.Len())
		om.Set("foo", "fooval")
		s.Require().True(om.Has("foo"))
		s.Require().Equal(1, om.Len())

		om.Delete("bar") // shouldn't cause any problems
		om.Delete("foo")
		s.Require().False(om.Has("foo"))
		s.Require().Equal(0, om.Len())
	})

	s.Run("multiple elements", func() {
		om := omap.SortedMap_String(map[string]string{
			"0": "w",
			"1": "x",
			"2": "y",
			"3": "z",
		})
		s.Require().Equal(4, om.Len())

		om.Set("11", "xx")
		om.Set("22", "yy")
		s.Require().Equal([]string{"0", "1", "11", "2", "22", "3"}, om.Keys())

		om.Delete("2") // shouldn't cause any problems
		s.Require().False(om.Has("2"))
		s.Require().Equal(5, om.Len())

		s.Require().Equal([]string{"0", "1", "11", "22", "3"}, om.Keys())
		om.Union(map[string]string{"222": "", "111": ""})
		s.Require().Equal([]string{"0", "1", "11", "111", "22", "222", "3"}, om.Keys())

		for k, v := range om.Data() {
			gotVal, exists := om.Get(k)
			s.True(exists)
			s.Require().Equal(v, gotVal)
		}
	})
}

// DummyValue is a blank struct to use as a placeholder in the maps for the
// generic value argument.
type DummyValue struct{}

// TestPair tests an SortedMap where the key is an asset.Pair.
func (s *Suite) TestPair() {
	pairStrs := []string{
		"abc:xyz", "abc:abc", "aaa:bbb", "xyz:xyz", "bbb:ccc", "xyz:abc",
	}
	sortedKeyStrs := []string{}
	sortedKeyStrs = append(sortedKeyStrs, pairStrs...)
	sort.Strings(sortedKeyStrs)

	sortedKeys := asset.MustNewPairs(sortedKeyStrs...)
	pairs := asset.MustNewPairs(pairStrs...)

	unsortedMap := make(map[asset.Pair]DummyValue)
	for _, pair := range pairs {
		unsortedMap[pair] = DummyValue{}
	}

	om := omap.SortedMap_Pair(unsortedMap)
	s.Require().Equal(6, om.Len())
	s.Require().EqualValues(sortedKeys, om.Keys())
	s.Require().NotEqualValues(asset.PairsToStrings(sortedKeys), pairStrs)

	var pairsFromLoop []asset.Pair
	for pair := range om.Range() {
		pairsFromLoop = append(pairsFromLoop, pair)
	}
	s.Require().EqualValues(sortedKeys, pairsFromLoop)
	s.Require().NotEqualValues(pairsFromLoop, pairs)
}

func (s *Suite) TestEthAddress() {
	s.Run("basic sorting", func() {
		var sortedKeys []gethcommon.Address
		unsortedMap := make(map[gethcommon.Address]DummyValue)

		// Prepare unsorted test inputs
		omapKeyInt64s := []int64{1, 0, 4, 6, 3, 2, 5}
		var unsortedKeys []gethcommon.Address
		for _, i := range omapKeyInt64s {
			bigInt := big.NewInt(i)
			key := gethcommon.BigToAddress(bigInt)
			unsortedKeys = append(unsortedKeys, key)
			unsortedMap[key] = DummyValue{}
		}

		{
			for _, i := range []int64{0, 1, 2, 3, 4, 5, 6} {
				sortedKeys = append(sortedKeys, gethcommon.BigToAddress(big.NewInt(i)))
			}
		}

		// Use sorter Sort
		om := omap.SortedMap_EthAddress(unsortedMap)
		s.Require().EqualValues(sortedKeys, om.Keys())
		s.NotEqualValues(unsortedKeys, sortedKeys)
	})
}
