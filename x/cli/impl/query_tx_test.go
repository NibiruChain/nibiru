package impl

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/evm"
	evmrpc "github.com/NibiruChain/nibiru/v2/evm/rpc"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

func TestQueryEthereumTransaction(t *testing.T) {
	hash := gethcommon.HexToHash(realEVMHash)
	const result = `{
  "blockHash": null,
  "blockNumber": null,
  "from": "0x1111111111111111111111111111111111111111",
  "gas": "0x5208",
  "gasPrice": "0x1",
  "hash": "0x66ef47cf9bda9bfb9c5e465a44e9a825bf1d8f013ed9b4abaf6960c02e9d85fc",
  "input": "0x",
  "nonce": "0x7",
  "to": "0x2222222222222222222222222222222222222222",
  "transactionIndex": null,
  "value": "0x2a",
  "type": "0x0",
  "v": "0x1b",
  "r": "0x1",
  "s": "0x2"
}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "application/json", request.Header.Get("Content-Type"))

		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		var payload struct {
			Method string   `json:"method"`
			Params []string `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Equal(t, "eth_getTransactionByHash", payload.Method)
		require.Equal(t, []string{hash.Hex()}, payload.Params)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":` + result + `}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	raw, transaction, err := queryEthereumTransaction(context.Background(), server.URL, hash)
	require.NoError(t, err)
	require.JSONEq(t, result, string(raw))
	require.Equal(t, hash, transaction.Hash)
	require.Nil(t, transaction.BlockHash)
	require.Nil(t, transaction.TransactionIndex)
}

func TestQueryEthereumTransactionErrors(t *testing.T) {
	hash := gethcommon.HexToHash(realEVMHash)
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "not found",
			statusCode: http.StatusOK,
			body:       `{"jsonrpc":"2.0","id":1,"result":null}`,
			wantErr:    "no EVM transaction found with hash",
		},
		{
			name:       "rpc error",
			statusCode: http.StatusOK,
			body:       `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"unavailable"}}`,
			wantErr:    "EVM JSON-RPC error -32000: unavailable",
		},
		{
			name:       "http error",
			statusCode: http.StatusBadGateway,
			body:       "upstream unavailable",
			wantErr:    "EVM JSON-RPC request returned HTTP 502",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, err := w.Write([]byte(tt.body))
				require.NoError(t, err)
			}))
			defer server.Close()

			raw, transaction, err := queryEthereumTransaction(context.Background(), server.URL, hash)
			require.Nil(t, raw)
			require.Nil(t, transaction)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestEVMRPCEndpoint(t *testing.T) {
	t.Run("uses explicit endpoint", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String(flagEVMRPC, "", "")
		require.NoError(t, cmd.Flags().Set(flagEVMRPC, "http://localhost:8545"))

		endpoint, err := evmRPCEndpoint(&cmd, client.Context{}.WithChainID("unknown"))
		require.NoError(t, err)
		require.Equal(t, "http://localhost:8545", endpoint)
	})

	t.Run("uses recognized network", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String(flagEVMRPC, "", "")

		endpoint, err := evmRPCEndpoint(&cmd, client.Context{}.WithChainID(appconst.SDK_CHAIN_ID_MAINNET))
		require.NoError(t, err)
		require.Equal(t, publicEVMRPCEndpoints[appconst.SDK_CHAIN_ID_MAINNET], endpoint)
	})

	t.Run("requires override for unknown network", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String(flagEVMRPC, "", "")

		_, err := evmRPCEndpoint(&cmd, client.Context{}.WithChainID("unknown"))
		require.EqualError(t, err, `no EVM JSON-RPC endpoint is configured for chain "unknown"; provide --evm-rpc`)
	})
}

func TestEVMRPCTransactionSummaryPending(t *testing.T) {
	hash := gethcommon.HexToHash(realEVMHash)

	pending := evmRPCTransactionSummary(&evmrpc.EthTxJsonRPC{Hash: hash})
	require.True(t, pending.Pending)
	require.Equal(t, realEVMHash, pending.EthHash)
	require.Empty(t, pending.TxHash)
	pendingJSON, err := json.Marshal(pending)
	require.NoError(t, err)
	var pendingFields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(pendingJSON, &pendingFields))
	require.NotContains(t, pendingFields, "gas_wanted")
	require.NotContains(t, pendingFields, "gas_used")

	blockHash := gethcommon.HexToHash("0x1234")
	committed := evmRPCTransactionSummary(&evmrpc.EthTxJsonRPC{
		Hash:      hash,
		BlockHash: &blockHash,
	})
	require.False(t, committed.Pending)
}

func TestTransactionQueryGasZeroIsPresent(t *testing.T) {
	zero := "0"
	output, err := json.Marshal(TransactionSummary{GasWanted: &zero, GasUsed: &zero})
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(output, &fields))
	require.JSONEq(t, `"0"`, string(fields["gas_wanted"]))
	require.JSONEq(t, `"0"`, string(fields["gas_used"]))
}

func TestTransactionQueryOutputFields(t *testing.T) {
	height, gasWanted, gasUsed := "46676744", "261408", "192897"
	summary := TransactionSummary{
		TxHash:    "COMET_HASH",
		EthHash:   "",
		Height:    &height,
		GasWanted: &gasWanted,
		GasUsed:   &gasUsed,
		Data:      "122E0A2C",
		Info:      "",
		Messages:  []string{"/cosmwasm.wasm.v1.MsgExecuteContract"},
		RawLog:    "execution failed",
		Logs:      json.RawMessage(`[{"msg_index":0,"log":"","events":[]}]`),
		Events:    json.RawMessage(`[{"type":"tx","attributes":[]}]`),
		Tx:        json.RawMessage(`{"@type":"/cosmos.tx.v1beta1.Tx","body":{"messages":[]}}`),
		EVM:       &EVMTransactionSummary{Hash: gethcommon.HexToHash(realEVMHash)},
	}
	output, err := json.Marshal(summary)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(output, &fields))
	require.JSONEq(t, `"COMET_HASH"`, string(fields["txhash"]))
	require.JSONEq(t, `""`, string(fields["eth_hash"]))
	require.JSONEq(t, `["/cosmwasm.wasm.v1.MsgExecuteContract"]`, string(fields["messages"]))
	require.JSONEq(t, `"execution failed"`, string(fields["raw_log"]))
	require.JSONEq(t, `"46676744"`, string(fields["height"]))
	require.JSONEq(t, `"261408"`, string(fields["gas_wanted"]))
	require.JSONEq(t, `"192897"`, string(fields["gas_used"]))
	require.JSONEq(t, `"122E0A2C"`, string(fields["data"]))
	require.JSONEq(t, `""`, string(fields["info"]))
	require.NotContains(t, fields, "codespace")
	require.JSONEq(t, `[{"msg_index":0,"log":"","events":[]}]`, string(fields["logs"]))
	require.JSONEq(t, `[{"type":"tx","attributes":[]}]`, string(fields["events"]))
	require.JSONEq(t, `{"@type":"/cosmos.tx.v1beta1.Tx","body":{"messages":[]}}`, string(fields["tx"]))
	require.NotContains(t, fields, "query_hash")
	require.NotContains(t, fields, "comet_hash")
	require.NotContains(t, fields, "evm_hash")

	details := transactionDetails{
		TransactionSummary: summary,
		Comet:              json.RawMessage(`{"txhash":"COMET_HASH"}`),
		EVMRPC:             &evmrpc.EthTxJsonRPC{Hash: gethcommon.HexToHash(realEVMHash)},
	}
	output, err = json.Marshal(details)
	require.NoError(t, err)
	var detailFields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(output, &detailFields))
	for key, value := range fields {
		require.JSONEq(t, string(value), string(detailFields[key]), key)
	}
	require.JSONEq(t, `{"txhash":"COMET_HASH"}`, string(detailFields["comet"]))
	require.Contains(t, detailFields, "evm_rpc")
	require.Contains(t, detailFields, "evm")
	require.NotContains(t, detailFields, "summary")
	require.NotContains(t, detailFields, "cosmos")
}

func TestDescribeTransactionMessagesIncludesEVMHash(t *testing.T) {
	evmMsg := evm.NewTx(&evm.EvmTxArgs{Nonce: 7, GasLimit: 21_000, GasPrice: big.NewInt(1)})
	messages, ethHash, err := describeTransactionMessages([]sdk.Msg{evmMsg})
	require.NoError(t, err)
	require.Equal(t, []string{sdk.MsgTypeURL(evmMsg)}, messages)
	require.Equal(t, evmMsg.AsTransaction().Hash().Hex(), ethHash)
}
