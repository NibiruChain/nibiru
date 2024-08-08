package omap_test

import (
	"fmt"
	"math/big"
	"sort"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/omap"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/precompile"
)

type Suite struct {
	suite.Suite
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(Suite))
}

// TestLenHasKeys checks the length of the ordered map and verifies if the map
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
			om := omap.OrderedMap_String[int](tc.dataMap)

			s.Require().Equal(tc.len, om.Len())

			orderedKeys := om.Keys()
			definitelyOrderedKeys := []string{}
			definitelyOrderedKeys = append(definitelyOrderedKeys, orderedKeys...)
			sort.Strings(definitelyOrderedKeys)

			s.Equal(definitelyOrderedKeys, orderedKeys)

			idx := 0
			for key := range om.Range() {
				s.Equal(orderedKeys[idx], key)
				idx++
			}
		})
	}
}

// TestGetSetDelete checks the Get, Set, and Delete operations on the OrderedMap.
func (s *Suite) TestGetSetDelete() {
	om := omap.OrderedMap_String[string](make(map[string]string))
	s.Require().Equal(0, om.Len())

	om.Set("foo", "fooval")
	s.Require().True(om.Has("foo"))
	s.Require().Equal(1, om.Len())

	om.Delete("bar") // shouldn't cause any problems
	om.Delete("foo")
	s.Require().False(om.Has("foo"))
	s.Require().Equal(0, om.Len())
}

// DummyValue is a blank struct to use as a placeholder in the maps for the
// generic value argument.
type DummyValue struct{}

// TestPair tests an OrderedMap where the key is an asset.Pair, a
// type that isn't built-in.
func (s *Suite) TestPair() {
	pairStrs := []string{
		"abc:xyz", "abc:abc", "aaa:bbb", "xyz:xyz", "bbb:ccc", "xyz:abc",
	}
	orderedKeyStrs := []string{}
	orderedKeyStrs = append(orderedKeyStrs, pairStrs...)
	sort.Strings(orderedKeyStrs)

	orderedKeys := asset.MustNewPairs(orderedKeyStrs...)
	pairs := asset.MustNewPairs(pairStrs...)

	unorderedMap := make(map[asset.Pair]DummyValue)
	for _, pair := range pairs {
		unorderedMap[pair] = DummyValue{}
	}

	om := omap.OrderedMap_Pair[DummyValue](unorderedMap)
	s.Require().Equal(6, om.Len())
	s.Require().EqualValues(orderedKeys, om.Keys())
	s.Require().NotEqualValues(asset.PairsToStrings(orderedKeys), pairStrs)

	var pairsFromLoop []asset.Pair
	for pair := range om.Range() {
		pairsFromLoop = append(pairsFromLoop, pair)
	}
	s.Require().EqualValues(orderedKeys, pairsFromLoop)
	s.Require().NotEqualValues(pairsFromLoop, pairs)
}

func (s *Suite) TestEthAddress() {
	s.Run("basic sorting", func() {
		var orderedKeys []gethcommon.Address
		unorderedMap := make(map[gethcommon.Address]DummyValue)

		// Prepare unsorted test inputs
		omapKeyInt64s := []int64{1, 0, 4, 6, 3, 2, 5}
		var unorderedKeys []gethcommon.Address
		for _, i := range omapKeyInt64s {
			bigInt := big.NewInt(i)
			key := gethcommon.BigToAddress(bigInt)
			unorderedKeys = append(unorderedKeys, key)
			unorderedMap[key] = DummyValue{}
		}

		{
			for _, i := range []int64{0, 1, 2, 3, 4, 5, 6} {
				orderedKeys = append(orderedKeys, gethcommon.BigToAddress(big.NewInt(i)))
			}
		}

		// Use sorter Sort
		om := omap.OrderedMap_EthAddress[DummyValue](unorderedMap)
		s.Require().EqualValues(orderedKeys, om.Keys())
		s.NotEqualValues(unorderedKeys, orderedKeys)
	})

	// This test proves that:
	// 1. The VM precompiles are ordered
	// 2. The output map from InitPrecompiles has the same addresses as the slice
	// of VM precompile addresses
	// 3. The ordered map produces the same ordered address slice.
	//
	// The VM precompiles are expected to be sorted. You'll notice from reading
	// the constants for precompile addresses imported from
	// "github.com/ethereum/go-ethereum/core/vm" that the order goes 1, 2, 3, ...
	s.Run("precompile address sorting", func() {
		// Check that the ordered map of precompiles is ordered
		deps := evmtest.NewTestDeps()
		// Types are written out to make the test easier to read.
		// Note that orderedKeys must be set after InitPrecompiles to mirror the
		// behavior of the Nibiru BaseApp.
		var unorderedMap map[gethcommon.Address]vm.PrecompiledContract
		unorderedMap = precompile.InitPrecompiles(deps.Chain.PublicKeepers)
		var orderedKeys []gethcommon.Address = vm.PrecompiledAddressesBerlin

		s.T().Log("1 | Prepare unsorted test inputs from output of \"precompile.InitPrecompiles\"")
		var unorderedKeys []gethcommon.Address
		for addr, _ := range unorderedMap {
			unorderedKeys = append(unorderedKeys, addr)
		}

		s.T().Log("2 | Compute ordered keys from VM")
		var vmAddrInts []*big.Int
		var vmAddrIntsBefore []*big.Int // unchanged copy of vmAddrInts
		for _, addr := range vm.PrecompiledAddressesBerlin {
			vmAddrInt := new(big.Int).SetBytes(addr.Bytes())
			vmAddrInts = append(vmAddrInts, vmAddrInt)
			vmAddrIntsBefore = append(vmAddrIntsBefore, vmAddrInt)
		}
		lessFunc := func(i, j int) bool {
			return vmAddrInts[i].Cmp(vmAddrInts[j]) < 0
		}
		sort.Slice(vmAddrInts, lessFunc)
		s.Require().EqualValues(vmAddrInts, vmAddrIntsBefore, "vm precompiles not ordered in InitPrecompiles")

		s.T().Log("3 | The ordered map produces the same ordered address slice")
		om := omap.OrderedMap_EthAddress[vm.PrecompiledContract](unorderedMap)
		s.Require().EqualValues(orderedKeys, om.Keys())
	})

}
