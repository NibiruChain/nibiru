package evm_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/evm"
	evmtest "github.com/NibiruChain/nibiru/x/evm/evmtest"

	"github.com/ethereum/go-ethereum/common"
)

func TestTransactionLogsValidate(t *testing.T) {
	addr := evmtest.NewEthAddr().String()

	testCases := []struct {
		name    string
		txLogs  evm.TransactionLogs
		expPass bool
	}{
		{
			"valid log",
			evm.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evm.Log{
					{
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
				},
			},
			true,
		},
		{
			"empty hash",
			evm.TransactionLogs{
				Hash: common.Hash{}.String(),
			},
			false,
		},
		{
			"nil log",
			evm.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evm.Log{nil},
			},
			false,
		},
		{
			"invalid log",
			evm.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evm.Log{{}},
			},
			false,
		},
		{
			"hash mismatch log",
			evm.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evm.Log{
					{
						Address:     addr,
						Topics:      []string{common.BytesToHash([]byte("topic")).String()},
						Data:        []byte("data"),
						BlockNumber: 1,
						TxHash:      common.BytesToHash([]byte("other_hash")).String(),
						TxIndex:     1,
						BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
						Index:       1,
						Removed:     false,
					},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.txLogs.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

func TestValidateLog(t *testing.T) {
	addr := evmtest.NewEthAddr().String()

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

func TestConversionFunctions(t *testing.T) {
	addr := evmtest.NewEthAddr().String()

	txLogs := evm.TransactionLogs{
		Hash: common.BytesToHash([]byte("tx_hash")).String(),
		Logs: []*evm.Log{
			{
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
		},
	}

	// convert valid log to eth logs and back (and validate)
	conversionLogs := evm.NewTransactionLogsFromEth(common.BytesToHash([]byte("tx_hash")), txLogs.EthLogs())
	conversionErr := conversionLogs.Validate()

	// create new transaction logs as copy of old valid one (and validate)
	copyLogs := evm.NewTransactionLogs(common.BytesToHash([]byte("tx_hash")), txLogs.Logs)
	copyErr := copyLogs.Validate()

	require.Nil(t, conversionErr)
	require.Nil(t, copyErr)
}
