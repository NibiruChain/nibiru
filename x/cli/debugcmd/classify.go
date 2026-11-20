// Package debugcmd implements `nibid debug`: decode, convert, and build
// encoded data. It infers what it was given when no subcommand is named.
package debugcmd

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/bech32"
)

// Kind names what an input was detected as.
type Kind string

const (
	// KindABCIQuery is a JSON-RPC body or batch (abci_query, broadcast_tx_*),
	// or the data of a known wrapper request such as a SimulateRequest.
	KindABCIQuery Kind = "abci-query"
	// KindAny is JSON with type_url and value fields, or a stargate wrapper.
	KindAny Kind = "any"
	// KindTx is a protobuf-encoded transaction in base64 or hex.
	KindTx Kind = "tx"
	// KindTxHash is a 32-byte transaction hash.
	KindTxHash Kind = "tx-hash"
	// KindAddr is a bech32, 0x EVM, or raw-hex account or contract address.
	KindAddr Kind = "addr"
	// KindPubkey is a proto-JSON pubkey, bech32 pubkey, or raw key bytes.
	KindPubkey Kind = "pubkey"
	// KindRawBytes is a byte list such as [10 21 13 255].
	KindRawBytes Kind = "raw-bytes"
	// KindProtobuf is binary data of no known shape yet. The decoder probes it
	// (tx, wrapper requests) and falls back to a raw field dump.
	KindProtobuf Kind = "protobuf"
	// KindUnknownProtobuf is valid protobuf with no known message type.
	KindUnknownProtobuf Kind = "unknown-protobuf"
	// KindAmbiguous means several kinds fit equally; see Candidates.
	KindAmbiguous Kind = "ambiguous"
	// KindUnknown is anything else.
	KindUnknown Kind = "unknown"
)

// Classification is the pure result of looking at an input's shape.
type Classification struct {
	Kind Kind
	// Candidates lists the kinds that fit when Kind is KindAmbiguous.
	Candidates []Kind
	// Payload holds the decoded bytes for binary and address inputs.
	Payload []byte
	// Encoding is "hex", "base64", "bech32", "evm", "json", or "".
	Encoding string
	// HRP is the bech32 human-readable part, when Encoding is "bech32".
	HRP string
}

var (
	rawBytesRe = regexp.MustCompile(`^\[\s*\d+(?:[\s,]+\d+)*\s*\]$`)
	bech32Re   = regexp.MustCompile(
		`^[a-z0-9]{1,83}1[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{6,}$`)
	hexRe = regexp.MustCompile(`^(?:0x)?(?:[0-9a-fA-F]{2})+$`)
)

// Classify inspects the shape of input and reports what it most likely is. It
// does no I/O and never decodes protobuf, so binary data that is not a known
// address or key length comes back as KindProtobuf for the caller to probe. A
// 32-byte value is ambiguous (transaction hash, Wasm contract address, or raw
// key) and is reported as KindAmbiguous with all candidates.
func Classify(input []byte) Classification {
	in := bytes.TrimSpace(input)
	s := string(in)
	if len(in) == 0 {
		return Classification{Kind: KindUnknown}
	}

	if rawBytesRe.MatchString(s) {
		return Classification{Kind: KindRawBytes, Encoding: "list"}
	}

	if (s[0] == '{' || s[0] == '[') && json.Valid(in) {
		return classifyJSON(in)
	}

	if bech32Re.MatchString(s) {
		if hrp, bz, err := bech32.DecodeAndConvert(s); err == nil {
			kind := KindAddr
			if strings.Contains(hrp, "pub") {
				kind = KindPubkey
			}
			return Classification{
				Kind: kind, Payload: bz, Encoding: "bech32", HRP: hrp,
			}
		}
	}

	if hexRe.MatchString(s) {
		bz, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
		if err == nil {
			if strings.HasPrefix(s, "0x") && len(bz) == 20 {
				return Classification{
					Kind: KindAddr, Payload: bz, Encoding: "evm",
				}
			}
			return classifyBytes(bz, "hex")
		}
	}

	if bz, err := base64.StdEncoding.DecodeString(s); err == nil && len(bz) > 0 {
		return classifyBytes(bz, "base64")
	}
	return Classification{Kind: KindUnknown}
}

func classifyBytes(bz []byte, encoding string) Classification {
	c := Classification{Payload: bz, Encoding: encoding}
	switch len(bz) {
	case 20:
		c.Kind = KindAddr
	case 32:
		c.Kind = KindAmbiguous
		c.Candidates = []Kind{KindTxHash, KindAddr, KindPubkey}
	case 33:
		c.Kind = KindPubkey
	default:
		c.Kind = KindProtobuf
	}
	return c
}

func classifyJSON(in []byte) Classification {
	var v any
	if err := json.Unmarshal(in, &v); err != nil {
		return Classification{Kind: KindUnknown}
	}
	c := Classification{Encoding: "json"}
	switch {
	case isJSONRPC(v):
		c.Kind = KindABCIQuery
	case isPubkeyJSON(v):
		c.Kind = KindPubkey
	case hasAnyShape(v):
		c.Kind = KindAny
	default:
		c.Kind = KindUnknown
	}
	return c
}

func isJSONRPC(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		method, _ := t["method"].(string)
		return method == "abci_query" || strings.HasPrefix(method, "broadcast_tx")
	case []any:
		if len(t) == 0 {
			return false
		}
		for _, item := range t {
			if !isJSONRPC(item) {
				return false
			}
		}
		return true
	}
	return false
}

func isPubkeyJSON(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	typ, _ := m["@type"].(string)
	_, hasKey := m["key"]
	return hasKey && strings.HasSuffix(typ, "PubKey")
}

// hasAnyShape reports whether v contains an object with "type_url" and "value"
// strings or a "stargate" key at any depth.
func hasAnyShape(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		if _, ok := t["stargate"]; ok {
			return true
		}
		if _, ok := t["type_url"].(string); ok {
			if _, ok := t["value"]; ok {
				return true
			}
		}
		for _, child := range t {
			if hasAnyShape(child) {
				return true
			}
		}
	case []any:
		for _, child := range t {
			if hasAnyShape(child) {
				return true
			}
		}
	}
	return false
}
