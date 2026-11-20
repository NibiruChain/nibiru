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
	// Readings holds every plausible interpretation of an ambiguous input, in
	// a fixed order.
	Readings Readings `json:"readings,omitempty"`
	// Error is set on a reading that failed to decode.
	Error string `json:"error,omitempty"`
}

// Reading is one named interpretation of an ambiguous input.
type Reading struct {
	Name  string
	Value any
}

// Readings is an ordered set of readings. It marshals to a JSON object whose
// keys keep the slice order, so output is deterministic.
type Readings []Reading

// MarshalJSON writes the readings as an object in slice order.
func (r Readings) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, rd := range r {
		if i > 0 {
			b.WriteByte(',')
		}
		k, err := json.Marshal(rd.Name)
		if err != nil {
			return nil, err
		}
		v, err := json.Marshal(rd.Value)
		if err != nil {
			return nil, err
		}
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
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
	if res, err, ok := decodeExplicitSchema(env, in, opts); ok {
		return res, err
	}
	c := Classify(in)
	kind := c.Kind
	if opts.As != "" {
		if res, err, ok := decodeAsBothEncodings(env, in, opts); ok {
			return res, err
		}
		kind = opts.As
		if c.Payload == nil {
			c = reclassifyAs(in, kind)
		}
	} else if res, ok := decodeBothEncodings(env, in); ok {
		return res, nil
	}
	return decodeKind(env, in, c, kind, opts)
}

// decodeKind decodes a classified input as kind.
func decodeKind(
	env Env, in []byte, c Classification, kind Kind, opts Options,
) (Result, error) {
	switch kind {
	case KindAmbiguous:
		return Result{
			Detected:   KindAmbiguous,
			Candidates: c.Candidates,
			Hint: fmt.Sprintf(
				"%d bytes fit several kinds; pass --as <kind> to choose",
				len(c.Payload)),
			Data:     map[string]any{"hex": hex.EncodeToString(c.Payload)},
			Readings: readings32(c.Payload),
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
	case KindAddr:
		out, err := DescribeAddr(c)
		return Result{Detected: KindAddr, Data: out}, err
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

// decodeExplicitSchema decodes input under a schema the caller named, before
// any heuristic runs: --type-url selects a message type, and --path selects a
// gRPC method for abci-query data. A failure is an error that names the
// schema, never a fall-through to guessing. It reports ok=false when no
// explicit schema applies. A JSON-RPC body keeps its own path and data, and an
// explicit --as other than protobuf (for --type-url) or abci-query (for
// --path) keeps the kind the caller chose.
func decodeExplicitSchema(
	env Env, in []byte, opts Options,
) (res Result, err error, ok bool) {
	switch {
	case opts.TypeURL != "" && (opts.As == "" || opts.As == KindProtobuf):
		bz, berr := binaryInput(in)
		if berr != nil {
			return Result{Detected: KindAny}, fmt.Errorf(
				"decode as %s: %w", opts.TypeURL, berr), true
		}
		out, derr := DecodeMsg(env, opts.TypeURL, bz)
		if derr != nil {
			return Result{Detected: KindAny}, fmt.Errorf(
				"decode as %s: %w", opts.TypeURL, derr), true
		}
		return Result{Detected: KindAny, Data: out}, nil, true
	case opts.Path != "" && (opts.As == "" || opts.As == KindABCIQuery):
		if len(in) > 0 && (in[0] == '{' || in[0] == '[') {
			return Result{}, nil, false
		}
		bz, berr := binaryInput(in)
		if berr != nil {
			return Result{Detected: KindABCIQuery}, fmt.Errorf(
				"decode as %s: %w", opts.Path, berr), true
		}
		out, derr := DecodeABCIRequest(env, opts.Path, bz, opts.Response)
		return Result{Detected: KindABCIQuery, Data: out}, derr, true
	}
	return Result{}, nil, false
}

// binaryInput reads hex (with optional 0x) or base64 protobuf bytes.
func binaryInput(in []byte) ([]byte, error) {
	s := strings.TrimSpace(string(in))
	if bz, err := hex.DecodeString(strings.TrimPrefix(s, "0x")); err == nil {
		return bz, nil
	}
	bz, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("input is neither hex nor base64")
	}
	return bz, nil
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

// AddrInfo is the offline description of an address.
type AddrInfo struct {
	Role    string            `json:"role"`
	Length  int               `json:"length"`
	Hex     string            `json:"hex"`
	EthAddr string            `json:"eth_addr,omitempty"`
	Bech32  map[string]string `json:"bech32"`
}

// DescribeAddr lists every form of an address: hex, the EVM 0x form, and
// the account, validator, and consensus bech32 forms. 20 bytes reads as an
// account and 32 bytes as a Wasm contract.
func DescribeAddr(c Classification) (AddrInfo, error) {
	bz := c.Payload
	info := AddrInfo{
		Length: len(bz),
		Hex:    hex.EncodeToString(bz),
		Bech32: map[string]string{},
	}
	switch len(bz) {
	case 20:
		info.Role = "account"
		info.EthAddr = gethcommon.BytesToAddress(bz).Hex()
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
	Type string            `json:"type"`
	Hex  string            `json:"hex"`
	Addr map[string]string `json:"addr"`
	// Alternatives lists other key types the same bytes could be.
	Alternatives []PubkeyInfo `json:"alternatives,omitempty"`
}

func describeKey(pk cryptotypes.PubKey) PubkeyInfo {
	addr := pk.Address()
	info := PubkeyInfo{
		Type: pk.Type(),
		Hex:  hex.EncodeToString(pk.Bytes()),
		Addr: map[string]string{
			"hex": hex.EncodeToString(addr),
			"acc": sdk.AccAddress(addr).String(),
			"val": sdk.ValAddress(addr).String(),
		},
	}
	if len(addr) == 20 && pk.Type() == ethsecp256k1.KeyType {
		info.Addr["eth_addr"] = gethcommon.BytesToAddress(addr).Hex()
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

// readings32 lists every reading of 32 bytes: a Wasm contract address, an
// ed25519 public key, and a transaction hash. All are computed offline.
func readings32(bz []byte) Readings {
	contract, _ := DescribeAddr(Classification{Payload: bz})
	key := describeKey(&ed25519.PubKey{Key: bz})
	key.Addr["cons"] = sdk.ConsAddress(consBytes(bz)).String()
	hash := strings.ToUpper(hex.EncodeToString(bz))
	return Readings{
		{Name: "contract_addr", Value: contract},
		{Name: "ed25519_pubkey", Value: key},
		{Name: "tx_hash", Value: map[string]any{
			"hash":  hash,
			"query": "nibiru query tx " + hash,
		}},
	}
}

// bothEncodings returns the hex and base64 readings of in when it is valid as
// both and only the hex reading is a generic-length payload (not a 20, 32, or
// 33 byte value, which keep their meaning, and not a transaction).
func bothEncodings(
	env Env, in []byte,
) (hexC, b64C Classification, ok bool) {
	s := string(in)
	if !hexRe.MatchString(s) || strings.HasPrefix(s, "0x") {
		return hexC, b64C, false
	}
	hb, err := hex.DecodeString(s)
	if err != nil {
		return hexC, b64C, false
	}
	bb, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(bb) == 0 {
		return hexC, b64C, false
	}
	hexC, b64C = classifyBytes(hb, "hex"), classifyBytes(bb, "base64")
	if hexC.Kind != KindProtobuf {
		return hexC, b64C, false
	}
	if _, err := DecodeTx(env, hb); err == nil {
		return hexC, b64C, false
	}
	if _, ok := probeWrapper(env, hb); ok {
		return hexC, b64C, false
	}
	return hexC, b64C, true
}

// decodeBothEncodings handles a string that is valid as both hex and base64:
// it returns one reading per encoding. A reading that does not decode shows
// its error.
func decodeBothEncodings(env Env, in []byte) (Result, bool) {
	hexC, b64C, ok := bothEncodings(env, in)
	if !ok {
		return Result{}, false
	}
	var rs Readings
	for _, rc := range []Classification{hexC, b64C} {
		res, err := decodeKind(env, in, rc, rc.Kind, Options{})
		if err != nil {
			res.Error = err.Error()
		}
		rs = append(rs, Reading{Name: rc.Encoding, Value: res})
	}
	return Result{
		Detected: KindAmbiguous,
		Hint: "the input is valid as hex and as base64; " +
			"each reading is shown",
		Readings: rs,
	}, true
}

// consBytes is the 20-byte address of an ed25519 key.
func consBytes(pub []byte) []byte {
	return (&ed25519.PubKey{Key: pub}).Address()
}

// dualBytes returns the hex and base64 decodings of in when it is valid as
// both. A 0x prefix means hex only.
func dualBytes(in []byte) (hb, bb []byte, ok bool) {
	s := string(in)
	if !hexRe.MatchString(s) || strings.HasPrefix(s, "0x") {
		return nil, nil, false
	}
	hb, err := hex.DecodeString(s)
	if err != nil {
		return nil, nil, false
	}
	bb, err = base64.StdEncoding.DecodeString(s)
	if err != nil || len(bb) == 0 {
		return nil, nil, false
	}
	return hb, bb, true
}

// kindFits reports whether n bytes can be a value of kind.
func kindFits(kind Kind, n int) bool {
	switch kind {
	case KindAddr:
		return n == 20 || n == 32
	case KindPubkey:
		return n == 32 || n == 33
	case KindTxHash:
		return n == 32
	}
	return true
}

// decodeAsBothEncodings applies an explicit --as kind to a string that is
// valid as both hex and base64. It tries each encoding and keeps the ones
// that decode as that kind. One match is the answer. Two matches are an
// ambiguous result with a hex and a base64 reading. No match is an error that
// names the kind.
func decodeAsBothEncodings(env Env, in []byte, opts Options) (Result, error, bool) {
	switch opts.As {
	case KindAny, KindAmbiguous, KindUnknown, KindUnknownProtobuf:
		return Result{}, nil, false
	}
	hb, bb, ok := dualBytes(in)
	if !ok {
		return Result{}, nil, false
	}
	var rs Readings
	var last Result
	for _, e := range []struct {
		name string
		bz   []byte
	}{{"hex", hb}, {"base64", bb}} {
		if !kindFits(opts.As, len(e.bz)) {
			continue
		}
		c := Classification{Kind: opts.As, Payload: e.bz, Encoding: e.name}
		res, err := decodeKind(env, in, c, opts.As, opts)
		if err != nil {
			continue
		}
		rs = append(rs, Reading{Name: e.name, Value: res})
		last = res
	}
	switch len(rs) {
	case 0:
		return Result{Detected: KindUnknown}, fmt.Errorf(
			"input does not decode as %s, read as hex or as base64", opts.As), true
	case 1:
		return last, nil, true
	}
	return Result{
		Detected: KindAmbiguous,
		Hint: fmt.Sprintf("the input is valid as hex and as base64, and both "+
			"decode as %s; each reading is shown", opts.As),
		Readings: rs,
	}, nil, true
}
