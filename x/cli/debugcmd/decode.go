package debugcmd

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/ed25519"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/bech32/legacybech32" //nolint:staticcheck // legacy pubkey bech32

	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"
)

// Env holds the codecs the decoders need. It is built from the client context
// of the running command, or from an app encoding config in tests.
type Env struct {
	Codec    codec.Codec
	Registry codectypes.InterfaceRegistry
	TxConfig client.TxConfig
	// DecodeAny decodes JSON containing type_url/value pairs (the decoder
	// shared with the legacy `base64-decode` command).
	DecodeAny func(jsonBz []byte) (any, error)
}

// Result is the JSON document `nibid debug` prints.
type Result struct {
	Detected   Kind   `json:"detected"`
	Candidates []Kind `json:"candidates,omitempty"`
	Hint       string `json:"hint,omitempty"`
	Data       any    `json:"data,omitempty"`
}

// Options carries the flags that steer decoding.
type Options struct {
	// As overrides the detected kind when non-empty.
	As Kind
	// TypeURL names the message type for raw protobuf input.
	TypeURL string
	// Path is the gRPC query path for raw abci-query data.
	Path string
	// Response decodes abci-query data as the response, not the request.
	Response bool
}

// Decode classifies input (unless opts.As is set) and decodes it.
func Decode(env Env, input []byte, opts Options) (Result, error) {
	in := bytes.TrimSpace(input)
	c := Classify(in)
	kind := c.Kind
	if opts.As != "" {
		kind = opts.As
		if c.Payload == nil {
			c = reclassifyAs(in, kind)
		}
	}

	switch kind {
	case KindAmbiguous:
		return Result{
			Detected:   KindAmbiguous,
			Candidates: c.Candidates,
			Hint: fmt.Sprintf(
				"%d bytes fit several kinds; pass --as <kind> to choose",
				len(c.Payload)),
			Data: map[string]any{"hex": hex.EncodeToString(c.Payload)},
		}, nil
	case KindABCIQuery:
		if c.Encoding == "json" {
			data, err := DecodeJSONRPC(env, in)
			return Result{Detected: KindABCIQuery, Data: data}, err
		}
		return decodeABCIData(env, c.Payload, opts)
	case KindAny:
		if env.DecodeAny == nil {
			return Result{}, errors.New("no any decoder configured")
		}
		out, err := env.DecodeAny(in)
		return Result{Detected: KindAny, Data: out}, err
	case KindTx:
		out, err := DecodeTx(env, c.Payload)
		return Result{Detected: KindTx, Data: out}, err
	case KindTxHash:
		return Result{
			Detected: KindTxHash,
			Data:     map[string]any{"hash": strings.ToUpper(hex.EncodeToString(c.Payload))},
			Hint:     "query it with: nibid query tx <hash>",
		}, nil
	case KindAddress:
		out, err := DescribeAddress(c)
		return Result{Detected: KindAddress, Data: out}, err
	case KindPubkey:
		out, err := DescribePubkey(env, in, c)
		return Result{Detected: KindPubkey, Data: out}, err
	case KindRawBytes:
		out, err := RawBytesToHex(string(in))
		return Result{Detected: KindRawBytes, Data: map[string]any{"hex": out}}, err
	case KindProtobuf, KindUnknownProtobuf:
		return decodeProtobuf(env, c.Payload, opts)
	}
	return Result{Detected: KindUnknown, Hint: "input is not recognized"},
		fmt.Errorf("could not classify input")
}

// reclassifyAs prepares a payload when --as forces a kind on an input that
// Classify did not decode (for example, forcing a hex string to "tx").
func reclassifyAs(in []byte, kind Kind) Classification {
	c := Classify(in)
	if c.Payload != nil {
		return c
	}
	s := strings.TrimPrefix(string(in), "0x")
	if bz, err := hex.DecodeString(s); err == nil {
		return Classification{Kind: kind, Payload: bz, Encoding: "hex"}
	}
	if bz, err := base64.StdEncoding.DecodeString(string(in)); err == nil {
		return Classification{Kind: kind, Payload: bz, Encoding: "base64"}
	}
	return Classification{Kind: kind, Encoding: c.Encoding}
}

// DecodeTx decodes protobuf transaction bytes to proto-JSON, the same form as
// `nibid tx decode`.
func DecodeTx(env Env, txBz []byte) (json.RawMessage, error) {
	tx, err := env.TxConfig.TxDecoder()(txBz)
	if err != nil {
		return nil, err
	}
	return env.TxConfig.TxJSONEncoder()(tx)
}

// DecodeMsg decodes protobuf bytes of the message named by typeURL to JSON.
func DecodeMsg(env Env, typeURL string, bz []byte) (json.RawMessage, error) {
	msg, err := env.Registry.Resolve(typeURL)
	if err != nil {
		return nil, fmt.Errorf("unknown type_url %q: %w", typeURL, err)
	}
	pm, ok := msg.(codec.ProtoMarshaler)
	if !ok {
		return nil, fmt.Errorf("type %q is not a protobuf message", typeURL)
	}
	if err := env.Codec.Unmarshal(bz, pm); err != nil {
		return nil, err
	}
	return env.Codec.MarshalJSON(pm)
}

func decodeProtobuf(env Env, bz []byte, opts Options) (Result, error) {
	if opts.TypeURL != "" {
		out, err := DecodeMsg(env, opts.TypeURL, bz)
		return Result{Detected: KindAny, Data: json.RawMessage(out)}, err
	}
	if opts.As == "" || opts.As == KindTx {
		if out, err := DecodeTx(env, bz); err == nil {
			return Result{Detected: KindTx, Data: out}, nil
		}
	}
	if opts.As == "" {
		if res, ok := probeWrapper(env, bz); ok {
			return res, nil
		}
	}
	if opts.As == KindTx {
		_, err := DecodeTx(env, bz)
		return Result{Detected: KindTx}, err
	}
	fields, err := RawFields(bz, 0)
	if err != nil {
		return Result{Detected: KindUnknown},
			fmt.Errorf("input is not valid protobuf: %w", err)
	}
	return Result{
		Detected: KindUnknownProtobuf,
		Hint: "valid protobuf of unknown type; pass --type-url " +
			"/pkg.Message to decode it",
		Data: fields,
	}, nil
}

// probeWrapper recognizes abci_query data that wraps a transaction
// (SimulateRequest field 2, BroadcastTxRequest field 1).
func probeWrapper(env Env, bz []byte) (Result, bool) {
	for _, path := range []string{
		"/cosmos.tx.v1beta1.Service/Simulate",
		"/cosmos.tx.v1beta1.Service/BroadcastTx",
	} {
		out, err := DecodeABCIRequest(env, path, bz, false)
		if err == nil && out.Nested != nil {
			return Result{Detected: KindABCIQuery, Data: out}, true
		}
	}
	return Result{}, false
}

func decodeABCIData(env Env, bz []byte, opts Options) (Result, error) {
	if opts.Path != "" {
		out, err := DecodeABCIRequest(env, opts.Path, bz, opts.Response)
		return Result{Detected: KindABCIQuery, Data: out}, err
	}
	res, ok := probeWrapper(env, bz)
	if ok {
		return res, nil
	}
	return Result{
			Detected: KindABCIQuery,
			Hint:     "pass --path /pkg.Service/Method to choose the request type",
		},
		errors.New("abci-query data needs --path unless it wraps a transaction")
}

// RawField is one field of a protobuf message decoded without a schema.
type RawField struct {
	Number int    `json:"field"`
	Wire   string `json:"wire"`
	Value  any    `json:"value"`
}

// RawFields lists the top-level fields of schemaless protobuf bytes. Bytes
// fields that are printable text show as strings, and those that parse as
// protobuf are expanded (to a small depth); others show as hex.
func RawFields(bz []byte, depth int) ([]RawField, error) {
	var out []RawField
	if len(bz) == 0 {
		return nil, errors.New("empty")
	}
	for len(bz) > 0 {
		num, typ, n := protowire.ConsumeTag(bz)
		if n < 0 || num < 1 {
			return nil, protowire.ParseError(n)
		}
		bz = bz[n:]
		f := RawField{Number: int(num)}
		switch typ {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(bz)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			f.Wire, f.Value, bz = "varint", v, bz[n:]
		case protowire.Fixed32Type:
			v, n := protowire.ConsumeFixed32(bz)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			f.Wire, f.Value, bz = "fixed32", v, bz[n:]
		case protowire.Fixed64Type:
			v, n := protowire.ConsumeFixed64(bz)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			f.Wire, f.Value, bz = "fixed64", v, bz[n:]
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(bz)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			f.Wire, f.Value, bz = "bytes", describeBytes(v, depth), bz[n:]
		default:
			return nil, fmt.Errorf("unsupported wire type %d", typ)
		}
		out = append(out, f)
	}
	return out, nil
}

func describeBytes(v []byte, depth int) any {
	if utf8.Valid(v) && isPrintable(v) {
		return string(v)
	}
	if depth < 3 {
		if nested, err := RawFields(v, depth+1); err == nil {
			return nested
		}
	}
	return "0x" + hex.EncodeToString(v)
}

func isPrintable(v []byte) bool {
	if len(v) == 0 {
		return false
	}
	for _, r := range string(v) {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// RawBytesToHex converts "[10 21 13 255]" to hex.
func RawBytesToHex(s string) (string, error) {
	s = strings.NewReplacer("[", " ", "]", " ", ",", " ").Replace(s)
	var bz []byte
	for _, f := range strings.Fields(s) {
		var n int
		if _, err := fmt.Sscanf(f, "%d", &n); err != nil || n < 0 || n > 255 {
			return "", fmt.Errorf("invalid byte %q", f)
		}
		bz = append(bz, byte(n))
	}
	return strings.ToUpper(hex.EncodeToString(bz)), nil
}

// AddressInfo is the offline description of an address.
type AddressInfo struct {
	Role   string            `json:"role"`
	Length int               `json:"length"`
	Hex    string            `json:"hex"`
	EVM    string            `json:"evm,omitempty"`
	Bech32 map[string]string `json:"bech32"`
}

// DescribeAddress lists every form of an address: hex, the EVM 0x form, and
// the account, validator, and consensus bech32 forms. 20 bytes reads as an
// account and 32 bytes as a Wasm contract.
func DescribeAddress(c Classification) (AddressInfo, error) {
	bz := c.Payload
	info := AddressInfo{
		Length: len(bz),
		Hex:    hex.EncodeToString(bz),
		Bech32: map[string]string{},
	}
	switch len(bz) {
	case 20:
		info.Role = "account"
		info.EVM = gethcommon.BytesToAddress(bz).Hex()
	case 32:
		info.Role = "wasm-contract"
	default:
		info.Role = "unknown"
	}
	if c.Encoding == "bech32" && c.HRP != "" {
		info.Bech32["input_hrp"] = c.HRP
	}
	info.Bech32["acc"] = sdk.AccAddress(bz).String()
	info.Bech32["val"] = sdk.ValAddress(bz).String()
	info.Bech32["cons"] = sdk.ConsAddress(bz).String()
	return info, nil
}

// PubkeyInfo is the offline description of a public key.
type PubkeyInfo struct {
	Type    string            `json:"type"`
	Hex     string            `json:"hex"`
	Address map[string]string `json:"address"`
	// Alternatives lists other key types the same bytes could be.
	Alternatives []PubkeyInfo `json:"alternatives,omitempty"`
}

func describeKey(pk cryptotypes.PubKey) PubkeyInfo {
	addr := pk.Address()
	info := PubkeyInfo{
		Type: pk.Type(),
		Hex:  hex.EncodeToString(pk.Bytes()),
		Address: map[string]string{
			"hex": hex.EncodeToString(addr),
			"acc": sdk.AccAddress(addr).String(),
			"val": sdk.ValAddress(addr).String(),
		},
	}
	if len(addr) == 20 && pk.Type() == ethsecp256k1.KeyType {
		info.Address["evm"] = gethcommon.BytesToAddress(addr).Hex()
	}
	return info
}

// DescribePubkey describes a proto-JSON pubkey, a bech32 pubkey, or raw key
// bytes. Raw 33-byte keys could be ethsecp256k1 (Nibiru's default) or
// secp256k1, so both are listed.
func DescribePubkey(env Env, in []byte, c Classification) (PubkeyInfo, error) {
	if c.Encoding == "json" {
		var pk cryptotypes.PubKey
		if err := env.Codec.UnmarshalInterfaceJSON(in, &pk); err != nil {
			return PubkeyInfo{}, err
		}
		return describeKey(pk), nil
	}
	if c.Encoding == "bech32" {
		for _, t := range []legacybech32.Bech32PubKeyType{
			legacybech32.AccPK, legacybech32.ValPK, legacybech32.ConsPK,
		} {
			if pk, err := legacybech32.UnmarshalPubKey(t, string(in)); err == nil { //nolint:staticcheck
				return describeKey(pk), nil
			}
		}
		return PubkeyInfo{}, fmt.Errorf("cannot parse bech32 pubkey %q", in)
	}
	switch len(c.Payload) {
	case secp256k1.PubKeySize:
		eth := describeKey(&ethsecp256k1.PubKey{Key: c.Payload})
		eth.Alternatives = []PubkeyInfo{
			describeKey(&secp256k1.PubKey{Key: c.Payload}),
		}
		return eth, nil
	case ed25519.PubKeySize:
		return describeKey(&ed25519.PubKey{Key: c.Payload}), nil
	}
	return PubkeyInfo{}, fmt.Errorf(
		"expected a 32 or 33 byte key, got %d bytes", len(c.Payload))
}
