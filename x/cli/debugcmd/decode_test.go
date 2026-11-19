package debugcmd_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"

	"github.com/NibiruChain/nibiru/v2/x/cli/debugcmd"
)

func testEnv() (debugcmd.Env, client.Context) {
	enc := app.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(enc.Codec).
		WithInterfaceRegistry(enc.InterfaceRegistry).
		WithTxConfig(enc.TxConfig).
		WithLegacyAmino(enc.Amino)
	return debugcmd.EnvFromContext(clientCtx, nil), clientCtx
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	bz, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return bz
}

func toJSON(t *testing.T, v any) string {
	t.Helper()
	bz, err := json.Marshal(v)
	require.NoError(t, err)
	return string(bz)
}

func TestDecodeAbciQueryArgFixture(t *testing.T) {
	env, _ := testEnv()
	data := strings.TrimSpace(string(readFixture(t, "abci_query_arg.hex")))

	// Detected from the bare hex: a SimulateRequest wrapping a tx.
	res, err := debugcmd.Decode(env, []byte(data), debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, debugcmd.KindABCIQuery, res.Detected)
	out := toJSON(t, res)
	require.Contains(t, out, "/cosmos.tx.v1beta1.Service/Simulate")
	require.Contains(t, out, "cosmwasm.wasm.v1.MsgExecuteContract")
	require.Contains(t, out, "queue_unbond")
}

func TestDecodeErisJSON(t *testing.T) {
	env, _ := testEnv()
	body := readFixture(t, "eris.json")
	res, err := debugcmd.Decode(env, body, debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, debugcmd.KindABCIQuery, res.Detected)

	entries, ok := res.Data.([]debugcmd.RPCEntry)
	require.True(t, ok)
	require.Len(t, entries, 11)
	paths := map[string]int{}
	for _, e := range entries {
		require.Empty(t, e.Error)
		require.NotNil(t, e.ABCI)
		require.NotEmpty(t, e.ABCI.Type, e.ABCI.Path)
		paths[e.ABCI.Path]++
	}
	require.Equal(t, 4, paths["/cosmos.tx.v1beta1.Service/Simulate"])
	require.Equal(t, 2, paths["/cosmwasm.wasm.v1.Query/SmartContractState"])

	first := toJSON(t, entries[0])
	require.Contains(t, first, "MsgExecuteContract")
	require.Contains(t, first, "tx_decoded")
	require.Contains(t, toJSON(t, entries), "queue_unbond")
	require.Contains(t, toJSON(t, entries[6]), "SmartContractState")
	// Query types decode with real field names.
	require.Contains(t, toJSON(t, entries[3]), `"address"`)
}

func TestDecodeSmartContractStateQuery(t *testing.T) {
	env, _ := testEnv()
	req := &wasmtypes.QuerySmartContractStateRequest{
		Address:   contract,
		QueryData: []byte(`{"config":{}}`),
	}
	bz, err := proto.Marshal(req)
	require.NoError(t, err)

	path := "/cosmwasm.wasm.v1.Query/SmartContractState"
	got, err := debugcmd.DecodeABCIRequest(env, path, bz, false)
	require.NoError(t, err)
	require.Equal(t, "cosmwasm.wasm.v1.QuerySmartContractStateRequest", got.Type)
	require.Contains(t, string(got.Message), contract)

	// Through a JSON-RPC body.
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "abci_query",
		"params": map[string]any{"path": path, "data": hex.EncodeToString(bz)},
	})
	require.NoError(t, err)
	res, err := debugcmd.Decode(env, body, debugcmd.Options{})
	require.NoError(t, err)
	entry, ok := res.Data.(debugcmd.RPCEntry)
	require.True(t, ok, "a single body yields one entry")
	require.Equal(t, got.Type, entry.ABCI.Type)

	// Response type.
	resp := &wasmtypes.QuerySmartContractStateResponse{Data: []byte(`{"a":1}`)}
	rbz, err := proto.Marshal(resp)
	require.NoError(t, err)
	got, err = debugcmd.DecodeABCIRequest(env, path, rbz, true)
	require.NoError(t, err)
	require.Equal(t, "cosmwasm.wasm.v1.QuerySmartContractStateResponse", got.Type)
}

func TestDecodeUnknownPath(t *testing.T) {
	env, _ := testEnv()
	// field 1 (bytes) = "abc"
	got, err := debugcmd.DecodeABCIRequest(env, "/store/bank/key", []byte{0x0a, 3, 'a', 'b', 'c'}, false)
	require.NoError(t, err)
	require.Contains(t, got.Note, "no registered gRPC method")
	require.Len(t, got.Fields, 1)
	require.Equal(t, "abc", got.Fields[0].Value)

	got, err = debugcmd.DecodeABCIRequest(env, "/no.Such/Method", []byte{0xff}, false)
	require.Error(t, err)
	_ = got
}

func TestDecodeUnknownProtobufHint(t *testing.T) {
	env, _ := testEnv()
	res, err := debugcmd.Decode(env, []byte("0a03616263"), debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, debugcmd.KindUnknownProtobuf, res.Detected)
	require.Contains(t, res.Hint, "--type-url")
}

func TestDecodeAddresses(t *testing.T) {
	env, _ := testEnv()

	res, err := debugcmd.Decode(env, []byte(acct), debugcmd.Options{})
	require.NoError(t, err)
	info, ok := res.Data.(debugcmd.AddressInfo)
	require.True(t, ok)
	require.Equal(t, "account", info.Role)
	require.Equal(t, acct, info.Bech32["acc"])
	require.Len(t, info.Hex, 40)
	require.True(t, strings.HasPrefix(info.EVM, "0x"))

	// EVM form round-trips to the same bech32 account.
	res, err = debugcmd.Decode(env, []byte(info.EVM), debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, acct, res.Data.(debugcmd.AddressInfo).Bech32["acc"])

	// Raw hex of the same 20 bytes too.
	res, err = debugcmd.Decode(env, []byte(info.Hex), debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, acct, res.Data.(debugcmd.AddressInfo).Bech32["acc"])

	// 32 bytes read as a Wasm contract.
	res, err = debugcmd.Decode(env, []byte(contract), debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, "wasm-contract", res.Data.(debugcmd.AddressInfo).Role)
}

func TestDecodeAmbiguous32Bytes(t *testing.T) {
	env, _ := testEnv()
	in := strings.Repeat("ab", 32)
	res, err := debugcmd.Decode(env, []byte(in), debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, debugcmd.KindAmbiguous, res.Detected)
	require.Len(t, res.Candidates, 3)

	res, err = debugcmd.Decode(env, []byte(in), debugcmd.Options{As: debugcmd.KindTxHash})
	require.NoError(t, err)
	require.Equal(t, debugcmd.KindTxHash, res.Detected)

	res, err = debugcmd.Decode(env, []byte(in), debugcmd.Options{As: debugcmd.KindAddress})
	require.NoError(t, err)
	require.Equal(t, "wasm-contract", res.Data.(debugcmd.AddressInfo).Role)
}

func TestDecodePubkeyAndRawBytes(t *testing.T) {
	env, _ := testEnv()
	res, err := debugcmd.Decode(env, []byte(
		`{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"AurroA7jvfPd1AadmmOvWM2rJSwipXfRf8yD6pLbA2DJ"}`),
		debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, debugcmd.KindPubkey, res.Detected)
	require.Equal(t, "secp256k1", res.Data.(debugcmd.PubkeyInfo).Type)

	// Raw 33 bytes: Nibiru's eth_secp256k1 first, with the Cosmos key as an
	// alternative.
	res, err = debugcmd.Decode(env, []byte("02"+strings.Repeat("11", 32)), debugcmd.Options{})
	require.NoError(t, err)
	info := res.Data.(debugcmd.PubkeyInfo)
	require.Equal(t, "eth_secp256k1", info.Type)
	require.Len(t, info.Alternatives, 1)

	res, err = debugcmd.Decode(env, []byte("[10 21 13 255]"), debugcmd.Options{})
	require.NoError(t, err)
	require.Equal(t, "0A150DFF", res.Data.(map[string]any)["hex"])
}

func TestDecodeAnyRequiresDecoder(t *testing.T) {
	env, _ := testEnv()
	_, err := debugcmd.Decode(env, []byte(`{"type_url":"/a.B","value":"AA=="}`), debugcmd.Options{})
	require.Error(t, err)
}
