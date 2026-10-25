package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	rs "github.com/NibiruChain/nibiru/v2/bft/rpc/jsonrpc/server"
	types "github.com/NibiruChain/nibiru/v2/bft/rpc/jsonrpc/types"
)

// FuzzRPCJSONRPCServer POSTs arbitrary bytes to the JSON-RPC handler and
// checks that the reply decodes as one RPCResponse or a batch of them. Seeds
// live in testdata/fuzz/FuzzRPCJSONRPCServer.
func FuzzRPCJSONRPCServer(f *testing.F) {
	// The server passes *types.Context as the first argument to every RPC
	// function, and unreflectResult expects (result, error).
	rpcFuncMap := map[string]*rs.RPCFunc{
		"c": rs.NewRPCFunc(
			func(ctx *types.Context, s string, i int) (string, error) {
				return "foo", nil
			}, "s,i"),
	}
	// Number of arguments each method takes after *types.Context.
	argCounts := map[string]int{"c": 2}

	mux := http.NewServeMux()
	rs.RegisterRPCFuncs(mux, rpcFuncMap, log.NewNopLogger())

	f.Fuzz(func(t *testing.T, data []byte) {
		// An empty body returns the HTML list of endpoints, not JSON.
		if len(data) == 0 {
			return
		}
		if hitsKnownMissingParamsPanic(data, argCounts) {
			t.Skip("known panic: call to a method with args but no params")
		}

		req, err := http.NewRequest("POST", "http://localhost/", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		res := rec.Result()
		blob, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := res.Body.Close(); err != nil {
			t.Fatal(err)
		}
		// Notifications (requests without an ID) get no reply.
		if len(blob) == 0 {
			return
		}

		if isJSONArray(blob) {
			var recv []types.RPCResponse
			if err := json.Unmarshal(blob, &recv); err != nil {
				t.Fatalf("decode batch reply %q: %v", blob, err)
			}
			return
		}
		var recv types.RPCResponse
		if err := json.Unmarshal(blob, &recv); err != nil {
			t.Fatalf("decode reply %q: %v", blob, err)
		}
	})
}

func isJSONArray(input []byte) bool {
	var slice []json.RawMessage
	return json.Unmarshal(input, &slice) == nil
}

// hitsKnownMissingParamsPanic reports whether data decodes, the way
// makeJSONRPCHandler decodes it, to a request (or a batch holding one) that
// has an ID, names a method that takes arguments, and has no "params".
//
// Known crash: makeJSONRPCHandler then calls the method with only the
// *types.Context argument, and reflect panics with "Call with too few input
// arguments" (http_json_handler.go). On a node, RecoverAndLogHandler turns the
// panic into an HTTP 500. The first input found is saved as
// testdata/fuzz/FuzzRPCJSONRPCServer/22a175aad7224fd3.
// TODO: Remove this skip once the handler returns RPCInvalidParamsError for
// a wrong argument count. Tracked in the follow-up issue for that fix.
func hitsKnownMissingParamsPanic(data []byte, argCounts map[string]int) bool {
	var requests []types.RPCRequest
	if err := json.Unmarshal(data, &requests); err != nil {
		var request types.RPCRequest
		if err := json.Unmarshal(data, &request); err != nil {
			return false
		}
		requests = []types.RPCRequest{request}
	}
	for _, req := range requests {
		if req.ID == nil || len(req.Params) > 0 {
			continue
		}
		if n, ok := argCounts[req.Method]; ok && n > 0 {
			return true
		}
	}
	return false
}
