package rpcapi_test

import (
	"math/big"

	gethmath "github.com/ethereum/go-ethereum/common/math"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

func (s *BackendSuite) TestChainID() {
	chainID, err := s.cli.EvmRpc.Eth.ChainId()
	s.Require().NoError(err)
	s.Require().Equal(appconst.ETH_CHAIN_ID_DEFAULT, chainID.ToInt().Int64())
}

func (s *BackendSuite) TestChainConfig() {
	config := s.backend.ChainConfig()
	s.Require().Equal(appconst.ETH_CHAIN_ID_DEFAULT, config.ChainID.Int64())
	s.Require().Equal(int64(0), config.LondonBlock.Int64())
}

func (s *BackendSuite) TestCurrentHeader() {
	currentHeader, err := s.backend.CurrentHeader()
	s.Require().NoError(err)
	s.Require().NotNil(currentHeader)
	s.Require().GreaterOrEqual(
		currentHeader.Number.Int64(),
		s.SuccessfulTxTransfer().BlockNumberRpc.Int64())
}

func (s *BackendSuite) TestPendingTransactions() {
	// Create pending tx: don't wait for next block
	randomEthAddr := evmtest.NewEthPrivAcc().EthAddr
	txHash := s.SendNibiViaEthTransfer(randomEthAddr, big.NewInt(123), false)
	txs, err := s.cli.EvmRpc.Eth.GetPendingTransactions()
	s.Require().NoError(err)
	s.Require().NotNil(txs)
	s.Require().NotNil(txHash)
	s.Require().Greater(len(txs), 0)
	txFound := false
	for _, tx := range txs {
		if tx.Hash == txHash {
			txFound = true
		}
	}
	s.Require().True(txFound, "pending tx not found")
}

func (s *BackendSuite) TestFeeHistory() {
	currentBlock, err := s.cli.EvmRpc.Eth.BlockNumber()
	s.Require().NoError(err)
	blockCount := 2 // blocks to search backwards from the current block
	percentiles := []float64{50, 100}

	res, err := s.cli.EvmRpc.Eth.FeeHistory(
		(gethmath.HexOrDecimal64)(blockCount),
		gethrpc.BlockNumber(int64(currentBlock)),
		percentiles,
	)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Reward, blockCount)
	s.Require().Len(res.BaseFee, blockCount+1)
	s.Require().Len(res.GasUsedRatio, len(percentiles))

	// Wallet zero-fee hint compatibility: https://github.com/NibiruChain/nibiru/pull/2601
	for _, baseFee := range res.BaseFee {
		s.Require().NotNil(baseFee)
		s.Require().Equal(evm.Big0, baseFee.ToInt())
	}
	for _, rewards := range res.Reward {
		for _, reward := range rewards {
			s.Require().NotNil(reward)
			s.Require().Equal(evm.Big0, reward.ToInt())
		}
	}

	for _, gasUsed := range res.GasUsedRatio {
		s.Require().LessOrEqual(gasUsed, float64(1))
	}
}

func (s *BackendSuite) TestSuggestGasTipCap() {
	tipCap, err := s.backend.SuggestGasTipCap(big.NewInt(1))
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(0), tipCap)
}

func (s *BackendSuite) TestGlobalMinGasPrice() {
	gasPrice, err := s.backend.GlobalMinGasPrice()
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(0), gasPrice)
}
