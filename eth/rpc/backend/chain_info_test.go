package backend_test

import (
	"math/big"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *BackendSuite) TestChainID() {
	s.Require().Equal(appconst.ETH_CHAIN_ID_DEFAULT, s.backend.ChainID().ToInt().Int64())
}

func (s *BackendSuite) TestChainConfig() {
	config := s.backend.ChainConfig()
	s.Require().Equal(appconst.ETH_CHAIN_ID_DEFAULT, config.ChainID.Int64())
	s.Require().Equal(int64(0), config.LondonBlock.Int64())
}

func (s *BackendSuite) TestBaseFeeWei() {
	resBlock, err := s.backend.TendermintBlockResultByNumber(transferTxBlockNumber.TmHeight())
	s.Require().NoError(err)
	baseFeeWei, err := s.backend.BaseFeeWei(resBlock)
	s.Require().NoError(err)
	s.Require().Equal(evm.BASE_FEE_WEI, baseFeeWei)
}

func (s *BackendSuite) TestCurrentHeader() {
	currentHeader, err := s.backend.CurrentHeader()
	s.Require().NoError(err)
	s.Require().NotNil(currentHeader)
	s.Require().GreaterOrEqual(currentHeader.Number.Int64(), transferTxBlockNumber.Int64())
}

func (s *BackendSuite) TestPendingTransactions() {
	// Create pending tx: don't wait for next block
	randomEthAddr := evmtest.NewEthPrivAcc().EthAddr
	txHash := s.SendNibiViaEthTransfer(randomEthAddr, big.NewInt(123), false)
	txs, err := s.backend.PendingTransactions()
	s.Require().NoError(err)
	s.Require().NotNil(txs)
	s.Require().NotNil(txHash)
	s.Require().Greater(len(txs), 0)
	txFound := false
	for _, tx := range txs {
		msg, err := evm.UnwrapEthereumMsg(tx, txHash)
		if err != nil {
			// not ethereum tx
			continue
		}
		if msg.Hash == txHash.String() {
			txFound = true
		}
	}
	s.Require().True(txFound, "pending tx not found")
}

func (s *BackendSuite) TestFeeHistory() {
	currentBlock, err := s.backend.BlockNumber()
	s.Require().NoError(err)
	blockCount := 2 // blocks to search backwards from the current block
	percentiles := []float64{50, 100}

	res, err := s.backend.FeeHistory(
		(gethrpc.DecimalOrHex)(blockCount),
		gethrpc.BlockNumber(int64(currentBlock)),
		percentiles,
	)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Reward, blockCount)
	s.Require().Len(res.BaseFee, blockCount+1)
	s.Require().Len(res.GasUsedRatio, len(percentiles))

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
