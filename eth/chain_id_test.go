package eth

import (
	"math/big"
	"strings"
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
			chainID: "nibiru_1-1",
			name:    "valid chain-id, single digit",
			expInt:  big.NewInt(1),
		},
		{
			chainID: "cataclysm_256-1",
			name:    "valid chain-id, multiple digits",
			expInt:  big.NewInt(256),
		},
		{
			chainID: "cataclysm-4-20",
			name:    "valid chain-id, dashed, multiple digits",
			expInt:  big.NewInt(4),
		},
		{
			chainID: "chain-1-1",
			name:    "valid chain-id, double dash",
			expInt:  big.NewInt(1),
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
		require.True(t, IsValidChainID(tc.chainID))
	}
}

func TestParseChainID_Sad(t *testing.T) {
	testCases := []struct {
		name    string
		chainID string
	}{
		{
			chainID: "chain_1_1",
			name:    "invalid chain-id, double underscore",
		},
		{
			chainID: "-",
			name:    "invalid chain-id, dash only",
		},
		{
			chainID: "-1",
			name:    "invalid chain-id, undefined identifier and EIP155",
		},
		{
			chainID: "_1-1",
			name:    "invalid chain-id, undefined identifier",
		},
		{
			chainID: "NIBIRU_1-1",
			name:    "invalid chain-id, uppercases",
		},
		{
			chainID: "Nibiru_1-1",
			name:    "invalid chain-id, mixed cases",
		},
		{
			chainID: "$&*#!_1-1",
			name:    "invalid chain-id, special chars",
		},
		{
			chainID: "nibiru_001-1",
			name:    "invalid eip155 chain-id, cannot start with 0",
		},
		{
			chainID: "nibiru_0x212-1",
			name:    "invalid eip155 chain-id, cannot invalid base",
		},
		{
			chainID: "nibiru_1-0x212",
			name:    "invalid eip155 chain-id, cannot invalid base",
		},
		{
			chainID: "nibiru_nibiru_9000-1",
			name:    "invalid eip155 chain-id, non-integer",
		},
		{
			chainID: "nibiru_-",
			name:    "invalid epoch, undefined",
		},
		{
			chainID: " ",
			name:    "blank chain ID",
		},
		{
			chainID: "",
			name:    "empty chain ID",
		},
		{
			chainID: "_-",
			name:    "empty content for chain id, eip155 and epoch numbers",
		},
		{
			chainID: "nibiru_" + strings.Repeat("1", 45) + "-1",
			name:    "long chain-id",
		},
	}

	for _, tc := range testCases {
		chainIDEpoch, err := ParseEthChainID(tc.chainID)
		require.Error(t, err, tc.name)
		require.Nil(t, chainIDEpoch)
		require.False(t, IsValidChainID(tc.chainID), tc.name)
	}
}
