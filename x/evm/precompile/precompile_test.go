package precompile_test

import (
	"math/big"
	"sort"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/common/omap"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

// This test proves that:
// 1. The VM precompiles are ordered
// 2. The output map from InitPrecompiles has the same addresses as the slice
// of VM precompile addresses
// 3. The ordered map produces the same ordered address slice.
//
// The VM precompiles are expected to be sorted. You'll notice from reading
// the constants for precompile addresses imported from
// "github.com/ethereum/go-ethereum/core/vm" that the order goes 1, 2, 3, ...
func (s *Suite) TestOrderedPrecompileAddresses() {
	s.T().Log("1 | Prepare test inputs from output of \"precompile.InitPrecompiles\"")
	// Types are written out to make the test easier to read.
	// Note that orderedKeys must be set after InitPrecompiles to mirror the
	// behavior of the Nibiru BaseApp.
	deps := evmtest.NewTestDeps()
	var unorderedMap map[gethcommon.Address]vm.PrecompiledContract = precompile.InitPrecompiles(deps.App.PublicKeepers)
	var orderedKeys []gethcommon.Address = vm.PrecompiledAddressesBerlin

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
	om := omap.SortedMap_EthAddress[vm.PrecompiledContract](unorderedMap)
	s.Require().EqualValues(orderedKeys, om.Keys())
}
