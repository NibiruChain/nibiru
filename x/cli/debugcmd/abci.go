package debugcmd

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx"
)

// ABCIResult is the decoded form of one gRPC query path plus its data.
type ABCIResult struct {
	Path string `json:"path"`
	// Type is the full name of the message that decoded the data.
	Type    string          `json:"type,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
	// Nested holds a transaction found inside a Simulate or BroadcastTx
	// request, decoded in full.
	Nested any `json:"tx_decoded,omitempty"`
	// Fields is the schemaless dump used when the path is unknown.
	Fields []RawField `json:"fields,omitempty"`
	Note   string     `json:"note,omitempty"`
}

// MethodTypes maps "/pkg.Service/Method" to the request and response message
// full names using the gogoproto service descriptors. It does not depend on
// the gRPC query router, which lacks some services at CLI time.
func MethodTypes(path string) (reqName, respName string, err error) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("%q is not a /pkg.Service/Method path", path)
	}
	desc, err := proto.HybridResolver.FindDescriptorByName(
		protoreflect.FullName(parts[0]))
	if err != nil {
		return "", "", fmt.Errorf("service %q not found: %w", parts[0], err)
	}
	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return "", "", fmt.Errorf("%q is not a gRPC service", parts[0])
	}
	method := svc.Methods().ByName(protoreflect.Name(parts[1]))
	if method == nil {
		return "", "", fmt.Errorf("service %q has no method %q", parts[0], parts[1])
	}
	return string(method.Input().FullName()), string(method.Output().FullName()), nil
}

func newMessage(fullName string) (proto.Message, error) {
	t := proto.MessageType(fullName)
	if t == nil {
		return nil, fmt.Errorf("message type %q is not registered", fullName)
	}
	msg, ok := reflect.New(t.Elem()).Interface().(proto.Message)
	if !ok {
		return nil, fmt.Errorf("type %q is not a protobuf message", fullName)
	}
	return msg, nil
}

// DecodeABCIRequest decodes the data of an abci_query for path. With response
// set, the data is decoded as the method's response instead of its request.
// An unknown path yields a raw field dump and an explanatory note.
func DecodeABCIRequest(
	env Env, path string, data []byte, response bool,
) (ABCIResult, error) {
	out := ABCIResult{Path: path}
	reqName, respName, err := MethodTypes(path)
	if err != nil {
		fields, ferr := RawFields(data, 0)
		out.Note = fmt.Sprintf(
			"no registered gRPC method for this path (%v); showing raw fields", err)
		if ferr != nil {
			return out, fmt.Errorf("%w; data is not valid protobuf: %v", err, ferr)
		}
		out.Fields = fields
		return out, nil
	}
	name := reqName
	if response {
		name = respName
	}
	msg, err := newMessage(name)
	if err != nil {
		return out, err
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return out, fmt.Errorf("decode as %s: %w", name, err)
	}
	out.Type = name
	out.Message, err = codec.ProtoMarshalJSON(msg, env.Registry)
	if err != nil {
		return out, err
	}

	var txBz []byte
	switch m := msg.(type) {
	case *tx.SimulateRequest:
		txBz = m.TxBytes
	case *tx.BroadcastTxRequest:
		txBz = m.TxBytes
	}
	if len(txBz) > 0 {
		nested, err := DecodeTx(env, txBz)
		if err != nil {
			return out, fmt.Errorf("decode nested tx_bytes: %w", err)
		}
		out.Nested = nested
	}
	return out, nil
}

// rpcRequest is a JSON-RPC request as the browser sends to a CometBFT node.
type rpcRequest struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
	Params struct {
		Path string `json:"path"`
		Data string `json:"data"`
		Tx   string `json:"tx"`
	} `json:"params"`
}

// RPCEntry is one decoded request of a JSON-RPC body.
type RPCEntry struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
	ABCI   *ABCIResult     `json:"abci_query,omitempty"`
	Tx     any             `json:"tx,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// DecodeJSONRPC decodes a JSON-RPC body or batch containing abci_query (hex
// params.data) and broadcast_tx_* (base64 params.tx) requests. A single body
// returns one RPCEntry and a batch returns a slice.
func DecodeJSONRPC(env Env, body []byte) (any, error) {
	body = []byte(strings.TrimSpace(string(body)))
	var reqs []rpcRequest
	batch := len(body) > 0 && body[0] == '['
	if batch {
		if err := json.Unmarshal(body, &reqs); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC batch: %w", err)
		}
	} else {
		var r rpcRequest
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC body: %w", err)
		}
		reqs = []rpcRequest{r}
	}

	entries := make([]RPCEntry, 0, len(reqs))
	var firstErr error
	var errs []error
	for _, r := range reqs {
		e := RPCEntry{ID: r.ID, Method: r.Method}
		var err error
		switch {
		case r.Method == "abci_query":
			var bz []byte
			bz, err = hex.DecodeString(strings.TrimPrefix(r.Params.Data, "0x"))
			if err != nil {
				err = fmt.Errorf("params.data is not hex: %w", err)
				break
			}
			var res ABCIResult
			res, err = DecodeABCIRequest(env, r.Params.Path, bz, false)
			e.ABCI = &res
		case strings.HasPrefix(r.Method, "broadcast_tx"):
			var bz []byte
			bz, err = base64.StdEncoding.DecodeString(r.Params.Tx)
			if err != nil {
				err = fmt.Errorf("params.tx is not base64: %w", err)
				break
			}
			e.Tx, err = DecodeTx(env, bz)
		default:
			err = fmt.Errorf("unsupported method %q", r.Method)
		}
		if err != nil {
			e.Error = err.Error()
			if batch {
				errs = append(errs, fmt.Errorf("entry %d (%s): %w",
					len(entries), r.Method, err))
			} else {
				firstErr = errors.New(e.Error)
			}
		}
		entries = append(entries, e)
	}
	if !batch {
		return entries[0], firstErr
	}
	return entries, errors.Join(errs...)
}
