package rpcapi_test

import (
	"encoding/json"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
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
			txHash:      s.SuccessfulTxTransfer().Receipt.TxHash,
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
			txResponse, err := s.cli.EvmRpc.Eth.GetTransactionByHash(tc.txHash)
			if !tc.wantTxFound {
				s.Require().Error(err)
				s.Require().Nil(txResponse)
				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(txResponse)
			s.Require().Equal(tc.txHash, txResponse.Hash)
			s.Require().Equal(s.evmSenderEthAddr, txResponse.From)
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
			txHash:      s.SuccessfulTxTransfer().Receipt.TxHash,
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
			txReceipt, err := s.cli.EvmRpc.Eth.GetTransactionReceipt(tc.txHash)

			if !tc.wantTxFound {
				s.Require().NoError(err)
				s.Require().Nil(txReceipt)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(txReceipt)

			// Check fields
			s.Equal(s.evmSenderEthAddr, txReceipt.From)
			s.Equal(&recipient, txReceipt.To)
			s.Greater(txReceipt.GasUsed, uint64(0))
			s.GreaterOrEqual(txReceipt.CumulativeGasUsed, txReceipt.GasUsed)
			s.Equal(tc.txHash, txReceipt.TxHash)
			s.Nil(txReceipt.ContractAddress)
			s.Require().Equal(gethcore.ReceiptStatusSuccessful, txReceipt.Status)
		})
	}
}

func (s *BackendSuite) TestGetTransactionLogs() {
	logs, err := s.cli.EvmRpc.Eth.GetTransactionLogs(
		s.SuccessfulTxDeployContract().Receipt.TxHash,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(logs)
	s.Require().Equal(
		s.SuccessfulTxDeployContract().Receipt.Logs,
		logs,
	)
	s.Require().Equal(
		*s.SuccessfulTxDeployContract().Receipt.ContractAddress,
		logs[0].Address,
	)
	s.Require().Equal(transferTopic(), logs[0].Topics[0])

	logs, err = s.cli.EvmRpc.Eth.GetTransactionLogs(
		gethcommon.BytesToHash([]byte("0x0")),
	)
	s.Require().ErrorContains(err, "tx not found")
	s.Require().Empty(logs)
}

func (s *BackendSuite) TestGetTransactionByBlockHashAndIndex() {
	blockWithTx, err := s.cli.EvmRpc.Eth.GetBlockByNumber(
		*s.SuccessfulTxTransfer().BlockNumberRpc, false)
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
			tx, err := s.cli.EvmRpc.Eth.GetTransactionByBlockHashAndIndex(
				tc.blockHash,
				hexutil.Uint(tc.txIndex),
			)

			if !tc.wantTxFound {
				s.Require().Nil(tx)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(tx)
			AssertTxResults(s, tx, s.SuccessfulTxTransfer().Receipt.TxHash)
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
			blockNumber: *s.SuccessfulTxTransfer().BlockNumberRpc,
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
			blockNumber: *s.SuccessfulTxTransfer().BlockNumberRpc,
			txIndex:     9999,
			wantTxFound: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx, err := s.cli.EvmRpc.Eth.GetTransactionByBlockNumberAndIndex(
				tc.blockNumber,
				hexutil.Uint(tc.txIndex),
			)

			if !tc.wantTxFound {
				s.Require().Nil(tx)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(tx)
			AssertTxResults(s, tx, s.SuccessfulTxTransfer().Receipt.TxHash)
		})
	}
}

func AssertTxResults(s *BackendSuite, tx *rpc.EthTxJsonRPC, expectedTxHash gethcommon.Hash) {
	s.Require().Equal(s.evmSenderEthAddr, tx.From)
	s.Require().Equal(&recipient, tx.To)
	s.Require().Greater(tx.Gas, uint64(0))
	s.Require().Equal(expectedTxHash, tx.Hash)
	s.Require().Equal(uint64(0), uint64(*tx.TransactionIndex))
}

func (s *BackendSuite) TestReceiptMarshalJson() {
	toAddr := evmtest.NewEthPrivAcc().EthAddr
	contractAddr := evmtest.NewEthPrivAcc().EthAddr
	tr := rpcapi.TransactionReceipt{
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
		ContractAddress:   &contractAddr,
		From:              evmtest.NewEthPrivAcc().EthAddr,
		To:                &toAddr,
		EffectiveGasPrice: (*hexutil.Big)(big.NewInt(1)),
	}

	jsonBz, err := tr.MarshalJSON()
	s.Require().NoError(err)

	gethReceipt := new(gethcore.Receipt)
	err = json.Unmarshal(jsonBz, gethReceipt)
	s.Require().NoError(err)

	receipt := new(rpcapi.TransactionReceipt)
	err = json.Unmarshal(jsonBz, receipt)
	s.Require().NoError(err)
	s.Require().Equal(tr.From, receipt.From)
	s.Require().Equal(tr.To, receipt.To)
	s.Require().Equal(tr.ContractAddress, receipt.ContractAddress)
	s.Require().Equal(tr.EffectiveGasPrice, receipt.EffectiveGasPrice)
}
