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

// TestAPIWeb3Sha3 pins the Keccak-256 result to a known vector so replacing
// Ethereum Keccak with standardized SHA3-256 would fail this regression.
func TestAPIWeb3Sha3(t *testing.T) {
	api := jsonrpc.NewImplWeb3API()

	got := api.Sha3(hexutil.Bytes([]byte("hello world")))
	want := hexutil.Bytes(hexutil.MustDecode("0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"))

	require.Equal(t, want, got)
}

// TestAPIWeb3Sha3JSONRPC checks that Ethereum DATA decodes into bytes before
// hashing. The cases catch hashing the encoded hex string and require malformed
// DATA to return an invalid-params error. Direct calls bypass parameter decoding.
func TestAPIWeb3Sha3JSONRPC(t *testing.T) {
	server := gethrpc.NewServer()
	require.NoError(t, server.RegisterName("web3", jsonrpc.NewImplWeb3API()))
	defer server.Stop()

	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	client, err := gethrpc.DialHTTP(httpServer.URL)
	require.NoError(t, err)
	defer client.Close()

	for _, tc := range []struct {
		name    string
		input   string
		decoded []byte
		invalid bool
	}{
		{name: "text bytes", input: "0x68656c6c6f20776f726c64", decoded: []byte("hello world")},
		{name: "empty bytes", input: "0x"},
		{name: "binary bytes", input: "0x00ff", decoded: []byte{0x00, 0xff}},
		{name: "missing prefix", input: "68656c6c6f", invalid: true},
		{name: "odd length", input: "0x0", invalid: true},
		{name: "invalid hex", input: "0xzz", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got hexutil.Bytes
			err := client.Call(&got, "web3_sha3", tc.input)
			if tc.invalid {
				require.Error(t, err)
				var rpcErr gethrpc.Error
				require.ErrorAs(t, err, &rpcErr)
				require.Equal(t, -32602, rpcErr.ErrorCode(), "malformed DATA must return invalid params")
				return
			}
			require.NoError(t, err)
			require.Equal(t, hexutil.Bytes(crypto.Keccak256(tc.decoded)), got)
		})
	}
}
