package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	types "github.com/NibiruChain/nibiru/v2/bft/rpc/jsonrpc/types"
)

func TestWebsocketManagerHandler(t *testing.T) {
	s := newWSServer()
	defer s.Close()

	// check upgrader works
	d := websocket.Dialer{}
	c, dialResp, err := d.Dial("ws://"+s.Listener.Addr().String()+"/websocket", nil)
	require.NoError(t, err)

	if got, want := dialResp.StatusCode, http.StatusSwitchingProtocols; got != want {
		t.Errorf("dialResp.StatusCode = %d, want %d", got, want)
	}

	// check basic functionality works
	req, err := types.MapToRequest(
		types.JSONRPCStringID("TestWebsocketManager"),
		"c",
		map[string]interface{}{"s": "a", "i": 10},
	)
	require.NoError(t, err)
	err = c.WriteJSON(req)
	require.NoError(t, err)

	var resp types.RPCResponse
	err = c.ReadJSON(&resp)
	require.NoError(t, err)
	require.Nil(t, resp.Error)
	dialResp.Body.Close()
}

// Calls that leave out arguments must get a reply with the request ID, and
// the connection must keep serving later calls.
func TestWebsocketMissingParams(t *testing.T) {
	s := newWSServer()
	defer s.Close()

	d := websocket.Dialer{}
	c, dialResp, err := d.Dial("ws://"+s.Listener.Addr().String()+"/websocket", nil)
	require.NoError(t, err)
	defer dialResp.Body.Close()
	defer c.Close()

	tests := []struct {
		payload  string
		wantCode int // 0 means no error
		id       types.JSONRPCIntID
	}{
		// no params at all works like {}
		{`{"jsonrpc": "2.0", "method": "c", "id": 1}`, 0, 1},
		// too few positional params
		{`{"jsonrpc": "2.0", "method": "c", "id": 2, "params": ["a"]}`, -32602, 2},
		// named params with missing keys use the zero value
		{`{"jsonrpc": "2.0", "method": "c", "id": 3, "params": {"s": "a"}}`, 0, 3},
		// a valid call still works afterwards
		{`{"jsonrpc": "2.0", "method": "c", "id": 4, "params": ["a", "10"]}`, 0, 4},
	}
	for i, tt := range tests {
		require.NoError(t, c.WriteMessage(websocket.TextMessage, []byte(tt.payload)), "#%d", i)

		var resp types.RPCResponse
		require.NoError(t, c.ReadJSON(&resp), "#%d", i)
		require.Equal(t, tt.id, resp.ID, "#%d: %v", i, resp)
		if tt.wantCode == 0 {
			require.Nil(t, resp.Error, "#%d", i)
		} else {
			require.NotNil(t, resp.Error, "#%d", i)
			require.Equal(t, tt.wantCode, resp.Error.Code, "#%d: %v", i, resp.Error)
		}
	}
}

func newWSServer() *httptest.Server {
	funcMap := map[string]*RPCFunc{
		"c": NewWSRPCFunc(func(ctx *types.Context, s string, i int) (string, error) { return "foo", nil }, "s,i"),
	}
	wm := NewWebsocketManager(funcMap)
	wm.SetLogger(log.TestingLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", wm.WebsocketHandler)

	return httptest.NewServer(mux)
}
