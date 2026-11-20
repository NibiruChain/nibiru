package debugcmd_test

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/NibiruChain/nibiru/v2/app" // sets bech32 prefixes
	"github.com/NibiruChain/nibiru/v2/x/cli/debugcmd"
)

const (
	acct     = "nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v"
	contract = "nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m"
)

func TestClassify(t *testing.T) {
	hex32 := strings.Repeat("ab", 32)
	hex20 := strings.Repeat("cd", 20)
	hex33 := "02" + strings.Repeat("11", 32)
	b64pb := base64.StdEncoding.EncodeToString([]byte{0x0a, 0x03, 'a', 'b', 'c'})

	for _, tc := range []struct {
		name       string
		in         string
		kind       debugcmd.Kind
		candidates []debugcmd.Kind
		encoding   string
	}{
		{"jsonrpc single", `{"jsonrpc":"2.0","id":1,"method":"abci_query","params":{"path":"/a.B/C","data":"00"}}`, debugcmd.KindABCIQuery, nil, "json"},
		{"jsonrpc batch", `[{"method":"abci_query"},{"method":"broadcast_tx_sync"}]`, debugcmd.KindABCIQuery, nil, "json"},
		{"jsonrpc other method", `{"method":"status"}`, debugcmd.KindUnknown, nil, "json"},
		{"any", `{"type_url":"/a.B","value":"AA=="}`, debugcmd.KindAny, nil, "json"},
		{"any nested", `{"x":{"type_url":"/a.B","value":"AA=="}}`, debugcmd.KindAny, nil, "json"},
		{"stargate wrapper", `{"stargate":{"type_url":"/a.B","value":"AA=="}}`, debugcmd.KindAny, nil, "json"},
		{"pubkey json", `{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"AurroA7jvfPd1AadmmOvWM2rJSwipXfRf8yD6pLbA2DJ"}`, debugcmd.KindPubkey, nil, "json"},
		{"bech32 account", acct, debugcmd.KindAddr, nil, "bech32"},
		{"bech32 contract", contract, debugcmd.KindAddr, nil, "bech32"},
		{"bech32 bad checksum", acct[:len(acct)-1] + "q", debugcmd.KindUnknown, nil, ""},
		{"evm address", "0x" + hex20, debugcmd.KindAddr, nil, "evm"},
		{"hex 20 bytes", hex20, debugcmd.KindAddr, nil, "hex"},
		{
			"hex 32 bytes ambiguous", hex32, debugcmd.KindAmbiguous,
			[]debugcmd.Kind{debugcmd.KindTxHash, debugcmd.KindAddr, debugcmd.KindPubkey},
			"hex",
		},
		{
			"0x 32 bytes ambiguous", "0x" + hex32, debugcmd.KindAmbiguous,
			[]debugcmd.Kind{debugcmd.KindTxHash, debugcmd.KindAddr, debugcmd.KindPubkey},
			"hex",
		},
		{"hex 33 byte key", hex33, debugcmd.KindPubkey, nil, "hex"},
		{"hex protobuf", "0a03616263", debugcmd.KindProtobuf, nil, "hex"},
		{"base64 protobuf", b64pb, debugcmd.KindProtobuf, nil, "base64"},
		{"raw bytes", "[10 21 13 255]", debugcmd.KindRawBytes, nil, "list"},
		{"raw bytes commas", "[1, 2, 3]", debugcmd.KindRawBytes, nil, "list"},
		{"padded whitespace", "  " + acct + "\n", debugcmd.KindAddr, nil, "bech32"},
		{"empty", "", debugcmd.KindUnknown, nil, ""},
		{"prose", "hello world", debugcmd.KindUnknown, nil, ""},
		{"invalid json", `{"a":`, debugcmd.KindUnknown, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := debugcmd.Classify([]byte(tc.in))
			require.Equal(t, tc.kind, got.Kind)
			require.Equal(t, tc.candidates, got.Candidates)
			if tc.encoding != "" {
				require.Equal(t, tc.encoding, got.Encoding)
			}
		})
	}
}

func TestClassifyPayloads(t *testing.T) {
	got := debugcmd.Classify([]byte("0x" + strings.Repeat("cd", 20)))
	require.Equal(t, strings.Repeat("cd", 20), hex.EncodeToString(got.Payload))

	got = debugcmd.Classify([]byte(acct))
	require.Len(t, got.Payload, 20)
	require.Equal(t, "nibi", got.HRP)

	got = debugcmd.Classify([]byte(contract))
	require.Len(t, got.Payload, 32)
}
