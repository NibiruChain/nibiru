package eth

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app/appconst"
)

func TestParseChainID_Happy(t *testing.T) {
	testCases := []struct {
		name    string
		chainID string
		expInt  *big.Int
	}{
		{
			chainID: "cataclysm-1",
			expInt:  big.NewInt(appconst.ETH_CHAIN_ID_MAINNET),
		},
		{
			chainID: "nibiru-localnet-0",
			name:    "valid nibiru-localnet-0",
			expInt:  big.NewInt(appconst.ETH_CHAIN_ID_LOCAL),
		},
		{
			chainID: "otherchain",
			name:    "other chain, default id",
			expInt:  big.NewInt(appconst.ETH_CHAIN_ID_DEFAULT),
		},
	}

	for _, tc := range testCases {
		chainIDEpoch, err := ParseEthChainIDStrict(tc.chainID)
		require.NoError(t, err, tc.name)
		var errMsg string = ""
		if err != nil {
			errMsg = err.Error()
		}
		assert.NoError(t, err, tc.name, errMsg)
		require.Equal(t, tc.expInt, chainIDEpoch, tc.name)
		require.True(t, IsValidChainID(tc.chainID))
	}
}
