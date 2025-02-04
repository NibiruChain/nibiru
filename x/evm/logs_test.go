package evm_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

	"github.com/ethereum/go-ethereum/common"
)

func TestValidateLog(t *testing.T) {
	addr := evmtest.NewEthPrivAcc().EthAddr.String()

	testCases := []struct {
		name    string
		log     *evm.Log
		expPass bool
	}{
		{
			"valid log",
			&evm.Log{
				Address:     addr,
				Topics:      []string{common.BytesToHash([]byte("topic")).String()},
				Data:        []byte("data"),
				BlockNumber: 1,
				TxHash:      common.BytesToHash([]byte("tx_hash")).String(),
				TxIndex:     1,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				Index:       1,
				Removed:     false,
			},
			true,
		},
		{
			"empty log", &evm.Log{}, false,
		},
		{
			"zero address",
			&evm.Log{
				Address: common.Address{}.String(),
			},
			false,
		},
		{
			"empty block hash",
			&evm.Log{
				Address:   addr,
				BlockHash: common.Hash{}.String(),
			},
			false,
		},
		{
			"zero block number",
			&evm.Log{
				Address:     addr,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				BlockNumber: 0,
			},
			false,
		},
		{
			"empty tx hash",
			&evm.Log{
				Address:     addr,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				BlockNumber: 1,
				TxHash:      common.Hash{}.String(),
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.log.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}
