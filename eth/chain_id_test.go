package eth

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChainID_Happy(t *testing.T) {
	testCases := []struct {
		name    string
		chainID string
		expInt  *big.Int
	}{
		{
			chainID: "cataclysm-1",
			expInt:  big.NewInt(100),
		},
		{
			chainID: "nibiru-localnet-0",
			name:    "valid nibiru-localnet-0",
			expInt:  big.NewInt(1000),
		},
		{
			chainID: "otherchain",
			name:    "other chain, default id",
			expInt:  big.NewInt(10000),
		},
	}

	for _, tc := range testCases {
		chainIDEpoch, err := ParseEthChainID(tc.chainID)
		require.NoError(t, err, tc.name)
		var errMsg string = ""
		if err != nil {
			errMsg = err.Error()
		}
		assert.NoError(t, err, tc.name, errMsg)
		require.Equal(t, tc.expInt, chainIDEpoch, tc.name)
	}
}
