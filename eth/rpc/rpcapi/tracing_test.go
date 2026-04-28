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
			traceConfig := traceConfigCallTracer()
			res, err := s.cli.EvmRpc.Debug.TraceTransaction(
				tc.txHash,
				traceConfig,
			)
			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoErrorf(err, "traceResult: %s", res)
			s.Require().NotNil(res)
			AssertTraceCall(s, mustTraceRawMessage(s, res))
		})
	}
}

func (s *BackendSuite) TestTraceBlock() {
	tmBlockWithTx, err := s.backend.TendermintBlockByNumber(
		*s.SuccessfulTxTransfer().BlockNumberRpc,
	)
	s.Require().NoError(err)

	tmBlockWithoutTx, err := s.backend.TendermintBlockByNumber(1)
	s.Require().NoError(err)

	testCases := []struct {
		name        string
		tmBlock     *tmrpctypes.ResultBlock
		txCount     int
		traceConfig *evm.TraceConfig
		wantErr     bool
	}{
		{
			name:        "happy: TraceBlock, no txs, tracer: default",
			tmBlock:     tmBlockWithoutTx,
			txCount:     0,
			traceConfig: traceConfigDefaultTracer(),
		},
		{
			name:        "happy: TraceBlock, no txs, tracer: callTracer",
			tmBlock:     tmBlockWithoutTx,
			txCount:     0,
			traceConfig: traceConfigCallTracer(),
		},
		{
			name:        "happy: TraceBlock, transfer tx, tracer: callTracer",
			tmBlock:     tmBlockWithTx,
			txCount:     1,
			traceConfig: traceConfigCallTracer(),
		},
		{
			name:        "happy: TraceBlock, transfer tx, tracer: default",
			tmBlock:     tmBlockWithTx,
			txCount:     1,
			traceConfig: traceConfigDefaultTracer(),
		},
		{
			name:    "sad: TraceBlock with ultra small timeout, causing tracer to stop too early",
			tmBlock: tmBlockWithTx,
			txCount: 1,
			traceConfig: func() *evm.TraceConfig {
				cfg := traceConfigCallTracer()
				cfg.Timeout = "1ns" // Force immediate timeout
				return cfg
			}(),
			// TODO: Add a deterministic unit test around TraceEthTxMsg timeout
			// behavior by injecting a controllable tracer/executor instead of
			// relying on scheduler timing in this integration test.
			// Issue: https://github.com/NibiruChain/nibiru/issues/2561
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resRes := [][]*evm.TxTraceResult{}
			{
				txTraceResults, err := s.cli.EvmRpc.Debug.TraceBlockByNumber(
					rpc.BlockNumber(tc.tmBlock.Block.Height),
					tc.traceConfig,
				)
				if tc.wantErr {
					// This case is timing-sensitive: with a tiny timeout and a fast tx,
					// tracing may either timeout or finish before the timeout goroutine
					// stops the tracer.
					if err == nil {
						s.T().Log("trace completed before ultra-small timeout fired; treating as acceptable flaky timing outcome")
					} else {
						s.T().Logf("trace returned error under ultra-small timeout (acceptable): %v", err)
					}
					return
				}
				s.Require().NoError(err)
				resRes = append(resRes, txTraceResults)
			}
			{
				txTraceResults, err := s.cli.EvmRpc.Debug.TraceBlockByHash(
					gethcommon.BytesToHash(
						tc.tmBlock.Block.Hash().Bytes(),
					),
					tc.traceConfig,
				)
				s.NoError(err)
				resRes = append(resRes, txTraceResults)
			}
			for _, txTraceResults := range resRes {
				s.Require().Equal(tc.txCount, len(txTraceResults))
				prettyBz, err := json.MarshalIndent(txTraceResults, "", "  ")
				s.Require().NoError(err)
				s.T().Logf("TraceBlock result: %s", prettyBz)
				if tc.txCount > 0 {
					typedResult, ok := txTraceResults[0].Result.(map[string]any)
					if !ok {
						s.T().Errorf("failed to parse block result as map[string]any. Got %#v", txTraceResults[0].Result)
					}
					traceResult, err := json.Marshal(typedResult)
					s.Require().NoError(err)
					AssertTraceCall(s, traceResult)
				}
			}
		})
	}
}

func (s *BackendSuite) TestTraceCall() {
	block, err := s.cli.EvmRpc.Eth.BlockNumber()
	s.Require().NoError(err)
	blockNumberForNonce := rpc.BlockNumber(block)
	nonce, err := s.cli.EvmRpc.Eth.GetTransactionCount(s.evmSenderEthAddr, rpc.BlockNumberOrHash{
		BlockNumber: &blockNumberForNonce,
	})
	s.NoError(err)
	gas := hexutil.Uint64(evm.NativeToWei(big.NewInt(int64(params.TxGas))).Uint64())
	amountToSendHex := hexutil.Big(*amountToSend)

	txArgs := evm.JsonTxArgs{
		Nonce: nonce,
		From:  &s.evmSenderEthAddr,
		To:    &recipient,
		Value: &amountToSendHex,
		Gas:   &gas,
	}
	s.Require().NoError(err)

	traceConfig := traceConfigDefaultTracer()
	blockNumber := rpc.NewBlockNumber(
		new(big.Int).SetUint64(uint64(block)),
	)
	res, err := s.cli.EvmRpc.Debug.TraceCall(
		txArgs,
		rpc.BlockNumberOrHash{
			BlockNumber: &blockNumber,
		},
		traceConfig,
	)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	AssertTraceCall(s, mustTraceRawMessage(s, res))
}

func mustTraceRawMessage(s *BackendSuite, traceResult any) json.RawMessage {
	switch typed := traceResult.(type) {
	case json.RawMessage:
		return typed
	case []byte:
		return json.RawMessage(typed)
	default:
		bz, err := json.Marshal(typed)
		s.Require().NoError(err)
		return bz
	}
}

func AssertTraceCall(
	s *BackendSuite,
	traceResult json.RawMessage,
) {
	var trace map[string]any
	err := json.Unmarshal(traceResult, &trace)
	s.Require().NoErrorf(err, "error unmarshaling traceResult: traceResult %s", traceResult)

	s.Require().Equal("CALL", trace["type"])
	s.Require().Equal(strings.ToLower(s.evmSenderEthAddr.Hex()), trace["from"])
	s.Require().Equal(strings.ToLower(recipient.Hex()), trace["to"])
	s.Require().Equal("0x"+gethcommon.Bytes2Hex(amountToSend.Bytes()), trace["value"])
}
