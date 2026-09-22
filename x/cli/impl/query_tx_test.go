package impl

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	evmrpc "github.com/NibiruChain/nibiru/v2/evm/rpc"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
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

	pending := evmRPCTransactionSummary(realEVMHash, &evmrpc.EthTxJsonRPC{Hash: hash})
	require.True(t, pending.Pending)

	blockHash := gethcommon.HexToHash("0x1234")
	committed := evmRPCTransactionSummary(realEVMHash, &evmrpc.EthTxJsonRPC{
		Hash:      hash,
		BlockHash: &blockHash,
	})
	require.False(t, committed.Pending)
}
