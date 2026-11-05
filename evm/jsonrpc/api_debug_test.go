package jsonrpc_test

import (
	"net/http/httptest"
	"testing"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	cmtrpcclient "github.com/NibiruChain/nibiru/v2/bft/rpc/jsonrpc/client"
	"github.com/NibiruChain/nibiru/v2/evm/jsonrpc"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
)

// TestDebugAPIOmitsGoRuntimeMethods registers the same debug service as the node
// and checks the JSON-RPC response. Profiling and process controls must be absent,
// rather than callable methods that return empty data or a compatibility error.
// This also catches accidentally reintroducing live memory and GC statistics.
func TestDebugAPIOmitsGoRuntimeMethods(t *testing.T) {
	apis := jsonrpc.GetRPCAPIs(
		server.NewDefaultContext(), client.Context{}, &cmtrpcclient.WSClient{},
		true, nil, []string{jsonrpc.NamespaceDebug},
	)
	require.Len(t, apis, 1)

	rpcServer := gethrpc.NewServer()
	require.NoError(t, rpcServer.RegisterName(apis[0].Namespace, apis[0].Service))
	defer rpcServer.Stop()

	httpServer := httptest.NewServer(rpcServer)
	defer httpServer.Close()
	rpcClient, err := gethrpc.DialHTTP(httpServer.URL)
	require.NoError(t, err)
	defer rpcClient.Close()

	for _, method := range []string{
		"debug_blockProfile", "debug_cpuProfile", "debug_gcStats", "debug_goTrace",
		"debug_memStats", "debug_setBlockProfileRate", "debug_stacks",
		"debug_writeBlockProfile", "debug_writeMemProfile", "debug_mutexProfile",
		"debug_setMutexProfileFraction", "debug_writeMutexProfile",
		"debug_freeOSMemory", "debug_setGCPercent",
		"debug_startCPUProfile", "debug_stopCPUProfile",
		"debug_startGoTrace", "debug_stopGoTrace",
	} {
		t.Run(method, func(t *testing.T) {
			var result any
			err := rpcClient.Call(&result, method)
			require.Error(t, err)
			var rpcErr gethrpc.Error
			require.ErrorAs(t, err, &rpcErr)
			require.Equal(t, -32601, rpcErr.ErrorCode(), "runtime method must not be registered")
		})
	}

	// Height zero fails before backend access. This verifies that the real EVM
	// trace method still dispatches without requiring a running local chain.
	t.Run("EVM tracing remains registered", func(t *testing.T) {
		var result any
		err := rpcClient.Call(&result, "debug_traceBlockByNumber", "0x0", nil)
		require.ErrorContains(t, err, "genesis is not traceable")
	})
}
