package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	types "github.com/NibiruChain/nibiru/v2/bft/rpc/jsonrpc/types"
)

// The URI handler builds one value per argument name, using the zero value
// for any that are missing, so leaving out query params is not an error.
func TestURIMissingParams(t *testing.T) {
	handler := RecoverAndLogHandler(testMux(), log.NewNopLogger())
	for i, url := range []string{
		"http://localhost/c",
		"http://localhost/c?s=%22a%22",
		"http://localhost/c?i=10",
		"http://localhost/block",
	} {
		req, _ := http.NewRequest("GET", url, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		res := rec.Result()
		blob, err := io.ReadAll(res.Body)
		require.NoError(t, err, "#%d", i)
		res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode, "#%d: %s", i, blob)
		recv := new(types.RPCResponse)
		require.NoError(t, json.Unmarshal(blob, recv), "#%d: %s", i, blob)
		assert.Nil(t, recv.Error, "#%d: %s", i, blob)
	}
}
