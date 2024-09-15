package backend_test

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
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

			s.Require().Equal(s.fundedAccEthAddr, receipt["from"])
			s.Require().Equal(&recipient, receipt["to"])
			s.Require().Greater(receipt["gasUsed"], uint64(0))
			s.Require().Equal(hexutil.Uint64(receipt["gasUsed"].(uint64)), receipt["cumulativeGasUsed"])
			s.Require().Equal(tc.txHash, receipt["transactionHash"])
			s.Require().Greater(receipt["transactionIndex"], hexutil.Uint64(0))
			s.Require().Nil(receipt["contractAddress"])
			s.Require().Equal(hexutil.Uint(gethcore.ReceiptStatusSuccessful), receipt["status"])
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
			txIndex:     1,
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
			txIndex:     1,
			wantTxFound: true,
		},
		{
			name:        "sad: block not found",
			blockNumber: rpc.NewBlockNumber(big.NewInt(9999999)),
			txIndex:     1,
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
	s.Require().Equal(uint64(1), uint64(*tx.TransactionIndex))
}
