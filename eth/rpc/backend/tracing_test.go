package backend_test

import (
	"math/big"
	"strings"

	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var traceConfig = &evm.TraceConfig{
	Tracer: "callTracer",
	TracerConfig: &evm.TracerConfig{
		OnlyTopCall: true,
	},
}

func (s *BackendSuite) TestTraceTransaction() {
	testCases := []struct {
		name    string
		txHash  gethcommon.Hash
		wantErr string
	}{
		{
			name:    "sad: tx not found",
			txHash:  gethcommon.BytesToHash([]byte("0x0")),
			wantErr: "not found",
		},
		{
			name:    "happy: tx found",
			txHash:  transferTxHash,
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := s.backend.TraceTransaction(
				tc.txHash,
				traceConfig,
			)
			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(res)
			AssertTraceCall(s, res.(map[string]interface{}))
		})
	}
}

func (s *BackendSuite) TestTraceBlock() {
	tmBlockWithTx, err := s.backend.TendermintBlockByNumber(transferTxBlockNumber)
	s.Require().NoError(err)

	blockNumberWithoutTx := rpc.NewBlockNumber(big.NewInt(1))
	tmBlockWithoutTx, err := s.backend.TendermintBlockByNumber(1)
	s.Require().NoError(err)

	testCases := []struct {
		name        string
		blockNumber rpc.BlockNumber
		tmBlock     *tmrpctypes.ResultBlock
		txCount     int
	}{
		{
			name:        "happy: block without txs",
			blockNumber: blockNumberWithoutTx,
			tmBlock:     tmBlockWithoutTx,
			txCount:     0,
		},
		{
			name:        "happy: block with txs",
			blockNumber: transferTxBlockNumber,
			tmBlock:     tmBlockWithTx,
			txCount:     1,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := s.backend.TraceBlock(
				tc.blockNumber,
				traceConfig,
				tc.tmBlock,
			)
			s.Require().NoError(err)
			s.Require().Equal(tc.txCount, len(res))
			if tc.txCount > 0 {
				AssertTraceCall(s, res[0].Result.(map[string]interface{}))
			}
		})
	}
}

func (s *BackendSuite) TestTraceCall() {
	block, err := s.backend.BlockNumber()
	s.Require().NoError(err)
	nonce, err := s.backend.GetTransactionCount(s.fundedAccEthAddr, rpc.BlockNumber(block))
	s.NoError(err)
	gas := hexutil.Uint64(evm.NativeToWei(big.NewInt(int64(params.TxGas))).Uint64())
	amountToSendHex := hexutil.Big(*amountToSend)

	txArgs := evm.JsonTxArgs{
		Nonce: nonce,
		From:  &s.fundedAccEthAddr,
		To:    &recipient,
		Value: &amountToSendHex,
		Gas:   &gas,
	}
	s.Require().NoError(err)

	res, err := s.backend.TraceCall(
		txArgs,
		rpc.BlockNumber(block),
		traceConfig,
	)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	AssertTraceCall(s, res.(map[string]interface{}))
}

func AssertTraceCall(s *BackendSuite, trace map[string]interface{}) {
	s.Require().Equal("CALL", trace["type"])
	s.Require().Equal(strings.ToLower(s.fundedAccEthAddr.Hex()), trace["from"])
	s.Require().Equal(strings.ToLower(recipient.Hex()), trace["to"])
	s.Require().Equal("0x"+gethcommon.Bytes2Hex(amountToSend.Bytes()), trace["value"])
}
