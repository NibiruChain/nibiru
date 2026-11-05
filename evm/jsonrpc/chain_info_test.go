package jsonrpc_test

import (
	"context"
	"fmt"
	"math/big"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethmath "github.com/ethereum/go-ethereum/common/math"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/evm/rpc"
	"github.com/NibiruChain/nibiru/v2/x/nutil/localnet"
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

// TestPendingTransactionLifecycle checks the public HTTP RPC after a real
// transfer. Receipt availability, rather than a fixed block increment, marks
// confirmation. The pending query may briefly lag that signal, so absence is
// checked with a bounded poll and transaction-specific diagnostics. A fast block
// can confirm before the first query; the controlled RPC snapshot test covers
// the genuinely unconfirmed phase without relying on that scheduling window.
func (s *BackendSuite) TestPendingTransactionLifecycle() {
	testMutex.Lock()
	defer testMutex.Unlock()

	rpcClient, err := gethrpc.DialHTTP(localnet.LocalnetEVMURI)
	s.Require().NoError(err)
	defer rpcClient.Close()

	txHash := s.SendNibiViaEthTransfer(evmtest.NewEthPrivAcc().EthAddr, big.NewInt(123), false)
	observedPending := false
	const lifecycleTimeout = 30 * time.Second
	lifecycleCtx, cancelLifecycle := context.WithTimeout(context.Background(), lifecycleTimeout)
	defer cancelLifecycle()
	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(lifecycleCtx, 2*time.Second)
		defer cancel()

		receipt, receiptErr := s.cli.EvmRpcClient.TransactionReceipt(ctx, txHash)
		var pending []*rpc.EthTxJsonRPC
		pendingErr := rpcClient.CallContext(ctx, &pending, "eth_getPendingTransactions")
		height, heightErr := s.cli.EvmRpcClient.BlockNumber(ctx)
		hashes := make([]gethcommon.Hash, 0, len(pending))
		for _, tx := range pending {
			if tx != nil {
				hashes = append(hashes, tx.Hash)
				observedPending = observedPending || tx.Hash == txHash
			}
		}
		diagnostic := fmt.Sprintf("tx=%s block_height=%d pending_count=%d receipt_present=%t receipt_error=%v pending_error=%v height_error=%v", txHash, height, len(pending), receipt != nil, receiptErr, pendingErr, heightErr)
		assert.NoError(c, receiptErr, diagnostic)
		assert.NoError(c, pendingErr, diagnostic)
		assert.NoError(c, heightErr, diagnostic)
		if !assert.NotNil(c, receipt, diagnostic) {
			return
		}
		assert.Equal(c, txHash, receipt.TxHash, diagnostic)
		assert.Equal(c, uint64(1), receipt.Status, diagnostic)
		if assert.NotNil(c, receipt.BlockNumber, diagnostic) {
			assert.Positive(c, receipt.BlockNumber.Sign(), diagnostic)
		}
		assert.NotContains(c, hashes, txHash, diagnostic)
	}, lifecycleTimeout, 100*time.Millisecond)
	s.T().Logf("confirmed tx %s is absent from pending RPC; observed_pending_before_confirmation=%t", txHash, observedPending)
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
