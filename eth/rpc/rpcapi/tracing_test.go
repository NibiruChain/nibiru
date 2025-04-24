package rpcapi_test

import (
	"encoding/json"
	"math/big"
	"strings"

	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func traceConfigCallTracer() *evm.TraceConfig {
	return &evm.TraceConfig{
		Tracer: "callTracer",
		TracerConfig: &evm.TracerConfig{
			OnlyTopCall: true,
		},
	}
}

func traceConfigDefaultTracer() *evm.TraceConfig {
	return &evm.TraceConfig{
		Tracer: "",
		TracerConfig: &evm.TracerConfig{
			OnlyTopCall: true,
		},
	}
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
			txHash:  s.SuccessfulTxTransfer().Receipt.TxHash,
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := s.backend.TraceTransaction(
				tc.txHash,
				traceConfigCallTracer(),
				// traceConfigDefaultTracer(),
			)
			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoErrorf(err, "traceResult: %s", res)
			s.Require().NotNil(res)
			AssertTraceCall(s, res)
		})
	}
}

func (s *BackendSuite) TestTraceBlock() {
	tmBlockWithTx, err := s.backend.TendermintBlockByNumber(
		*s.SuccessfulTxTransfer().BlockNumberRpc,
	)
	s.Require().NoError(err)

	blockNumberWithoutTx := rpc.NewBlockNumber(big.NewInt(1))
	tmBlockWithoutTx, err := s.backend.TendermintBlockByNumber(1)
	s.Require().NoError(err)

	testCases := []struct {
		name        string
		blockNumber rpc.BlockNumber
		tmBlock     *tmrpctypes.ResultBlock
		txCount     int
		traceConfig *evm.TraceConfig
	}{
		{
			name:        "happy: TraceBlock, no txs, tracer: default",
			blockNumber: blockNumberWithoutTx,
			tmBlock:     tmBlockWithoutTx,
			txCount:     0,
			traceConfig: traceConfigDefaultTracer(),
		},
		{
			name:        "happy: TraceBlock, no txs, tracer: callTracer",
			blockNumber: blockNumberWithoutTx,
			tmBlock:     tmBlockWithoutTx,
			txCount:     0,
			traceConfig: traceConfigCallTracer(),
		},
		{
			name:        "happy: TraceBlock, transfer tx, tracer: callTracer",
			blockNumber: *s.SuccessfulTxTransfer().BlockNumberRpc,
			tmBlock:     tmBlockWithTx,
			txCount:     1,
			traceConfig: traceConfigCallTracer(),
		},
		{
			name:        "happy: TraceBlock, transfer tx, tracer: default",
			blockNumber: *s.SuccessfulTxTransfer().BlockNumberRpc,
			tmBlock:     tmBlockWithTx,
			txCount:     1,
			traceConfig: traceConfigDefaultTracer(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := s.backend.TraceBlock(
				tc.blockNumber,
				tc.traceConfig,
				tc.tmBlock,
			)
			s.Require().NoError(err)
			s.Require().Equal(tc.txCount, len(res))
			prettyBz, err := json.MarshalIndent(res, "", "  ")
			s.Require().NoError(err)
			s.T().Logf("TraceBlock result: %s", prettyBz)
			if tc.txCount > 0 {
				typedResult, ok := res[0].Result.(map[string]any)
				if !ok {
					s.T().Errorf("failed to parse block result as map[string]any. Got %#v", res[0].Result)
				}
				traceResult, err := json.Marshal(typedResult)
				s.Require().NoError(err)
				AssertTraceCall(s, traceResult)
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

	traceConfig := traceConfigDefaultTracer()
	res, err := s.backend.TraceCall(
		txArgs,
		rpc.BlockNumber(block),
		traceConfig,
	)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	AssertTraceCall(s, res)
}

func AssertTraceCall(
	s *BackendSuite,
	traceResult json.RawMessage,
) {
	var trace map[string]any
	err := json.Unmarshal(traceResult, &trace)
	s.Require().NoErrorf(err, "error unmarshaling traceResult: traceResult %s", traceResult)

	s.Require().Equal("CALL", trace["type"])
	s.Require().Equal(strings.ToLower(s.fundedAccEthAddr.Hex()), trace["from"])
	s.Require().Equal(strings.ToLower(recipient.Hex()), trace["to"])
	s.Require().Equal("0x"+gethcommon.Bytes2Hex(amountToSend.Bytes()), trace["value"])
}
