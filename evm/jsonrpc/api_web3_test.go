package jsonrpc_test

import (
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/evm/jsonrpc"
)

func TestAPIWeb3Sha3(t *testing.T) {
	api := jsonrpc.NewImplWeb3API()

	got := api.Sha3(hexutil.Bytes([]byte("hello world")))
	want := hexutil.Bytes(hexutil.MustDecode("0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"))

	require.Equal(t, want, got)
}

func TestAPIWeb3Sha3JSONRPC(t *testing.T) {
	server := gethrpc.NewServer()
	require.NoError(t, server.RegisterName("web3", jsonrpc.NewImplWeb3API()))
	defer server.Stop()

	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	client, err := gethrpc.DialHTTP(httpServer.URL)
	require.NoError(t, err)
	defer client.Close()

	var got hexutil.Bytes
	require.NoError(t, client.Call(&got, "web3_sha3", "0x68656c6c6f20776f726c64"))

	want := crypto.Keccak256([]byte("hello world"))
	require.Equal(t, hexutil.Bytes(want), got)
}
