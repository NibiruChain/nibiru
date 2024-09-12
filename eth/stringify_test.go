package eth_test

import (
	"github.com/NibiruChain/nibiru/v2/eth"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
)

func (s *Suite) TestStringify() {
	testCases := []gethcore.Bloom{
		gethcore.BytesToBloom([]byte("alphanumeric123")),
		gethcore.BytesToBloom([]byte{}),
		gethcore.BytesToBloom(gethcommon.Big0.Bytes()),
		gethcore.BytesToBloom(gethcommon.Big1.Bytes()),
	}
	for tcIdx, bloom := range testCases {
		gotStr := eth.BloomToHex(bloom)
		gotBloom, err := eth.BloomFromHex(gotStr)
		s.NoError(err)
		s.Equalf(bloom, gotBloom, "test case: %d", tcIdx)
	}
}
