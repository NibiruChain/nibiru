package backend_test

import (
	"encoding/json"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *BackendSuite) TestGetTransactionByHash() {
	testCases := []struct {
		name        string
		txHash      gethcommon.Hash
		wantTxFound bool
	}{
		{
			name:        "happy: tx found",
			txHash:      transferTxHash,
			wantTxFound: true,
		},
		{
			name:        "sad: tx not found",
			txHash:      gethcommon.BytesToHash([]byte("0x0")),
			wantTxFound: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			txResponse, err := s.backend.GetTransactionByHash(tc.txHash)
			if !tc.wantTxFound {
				s.Require().Nil(txResponse)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(txResponse)
			s.Require().Equal(tc.txHash, txResponse.Hash)
			s.Require().Equal(s.fundedAccEthAddr, txResponse.From)
			s.Require().Equal(&recipient, txResponse.To)
			s.Require().Equal(amountToSend, txResponse.Value.ToInt())
		})
	}
}

func (s *BackendSuite) TestGetTransactionReceipt() {
	testCases := []struct {
		name        string
		txHash      gethcommon.Hash
		wantTxFound bool
	}{
		{
			name:        "happy: tx found",
			txHash:      transferTxHash,
			wantTxFound: true,
		},
		{
			name:        "sad: tx not found",
			txHash:      gethcommon.BytesToHash([]byte("0x0")),
			wantTxFound: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			receipt, err := s.backend.GetTransactionReceipt(tc.txHash)
			if !tc.wantTxFound {
				s.Require().Nil(receipt)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(receipt)

			// Check fields
			s.Equal(s.fundedAccEthAddr, receipt.From)
			s.Equal(&recipient, receipt.To)
			s.Greater(receipt.GasUsed, uint64(0))
			s.Equal(receipt.GasUsed, receipt.CumulativeGasUsed)
			s.Equal(tc.txHash, receipt.TxHash)
			s.Nil(receipt.ContractAddress)
			s.Require().Equal(gethcore.ReceiptStatusSuccessful, receipt.Status)
		})
	}
}

func (s *BackendSuite) TestGetTransactionByBlockHashAndIndex() {
	blockWithTx, err := s.backend.GetBlockByNumber(transferTxBlockNumber, false)
	s.Require().NoError(err)
	blockHash := gethcommon.BytesToHash(blockWithTx["hash"].(hexutil.Bytes))

	testCases := []struct {
		name        string
		blockHash   gethcommon.Hash
		txIndex     uint
		wantTxFound bool
	}{
		{
			name:        "happy: tx found",
			blockHash:   blockHash,
			txIndex:     0,
			wantTxFound: true,
		},
		{
			name:        "sad: block not found",
			blockHash:   gethcommon.BytesToHash([]byte("0x0")),
			txIndex:     1,
			wantTxFound: false,
		},
		{
			name:        "sad: tx not found",
			blockHash:   blockHash,
			txIndex:     9999,
			wantTxFound: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx, err := s.backend.GetTransactionByBlockHashAndIndex(tc.blockHash, hexutil.Uint(tc.txIndex))
			if !tc.wantTxFound {
				s.Require().Nil(tx)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(tx)
			AssertTxResults(s, tx, transferTxHash)
		})
	}
}

func (s *BackendSuite) TestGetTransactionByBlockNumberAndIndex() {
	testCases := []struct {
		name        string
		blockNumber rpc.BlockNumber
		txIndex     uint
		wantTxFound bool
	}{
		{
			name:        "happy: tx found",
			blockNumber: transferTxBlockNumber,
			txIndex:     0,
			wantTxFound: true,
		},
		{
			name:        "sad: block not found",
			blockNumber: rpc.NewBlockNumber(big.NewInt(9999999)),
			txIndex:     0,
			wantTxFound: false,
		},
		{
			name:        "sad: tx not found",
			blockNumber: transferTxBlockNumber,
			txIndex:     9999,
			wantTxFound: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx, err := s.backend.GetTransactionByBlockNumberAndIndex(tc.blockNumber, hexutil.Uint(tc.txIndex))
			if !tc.wantTxFound {
				s.Require().Nil(tx)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(tx)
			AssertTxResults(s, tx, transferTxHash)
		})
	}
}

func AssertTxResults(s *BackendSuite, tx *rpc.EthTxJsonRPC, expectedTxHash gethcommon.Hash) {
	s.Require().Equal(s.fundedAccEthAddr, tx.From)
	s.Require().Equal(&recipient, tx.To)
	s.Require().Greater(tx.Gas, uint64(0))
	s.Require().Equal(expectedTxHash, tx.Hash)
	s.Require().Equal(uint64(0), uint64(*tx.TransactionIndex))
}

func (s *BackendSuite) TestReceiptMarshalJson() {
	toAddr := evmtest.NewEthPrivAcc().EthAddr
	tr := backend.TransactionReceipt{
		Receipt: gethcore.Receipt{
			Type:              0,
			PostState:         []byte{},
			Status:            0,
			CumulativeGasUsed: 0,
			Bloom:             [256]byte{},
			Logs:              []*gethcore.Log{},
			TxHash:            [32]byte{},
			ContractAddress:   [20]byte{},
			GasUsed:           0,
			BlockHash:         [32]byte{},
			BlockNumber:       &big.Int{},
			TransactionIndex:  0,
		},
		ContractAddress:   nil,
		From:              evmtest.NewEthPrivAcc().EthAddr,
		To:                &toAddr,
		EffectiveGasPrice: (*hexutil.Big)(big.NewInt(1)),
	}

	jsonBz, err := tr.MarshalJSON()
	s.Require().NoError(err)

	gethReceipt := new(gethcore.Receipt)
	err = json.Unmarshal(jsonBz, gethReceipt)
	s.Require().NoError(err)

	receipt := new(backend.TransactionReceipt)
	err = json.Unmarshal(jsonBz, receipt)
	s.Require().NoError(err)
}
