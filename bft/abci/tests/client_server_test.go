package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abciclient "github.com/NibiruChain/nibiru/v2/bft/abci/client"
	"github.com/NibiruChain/nibiru/v2/bft/abci/example/kvstore"
	abciserver "github.com/NibiruChain/nibiru/v2/bft/abci/server"
	cmtnet "github.com/NibiruChain/nibiru/v2/bft/libs/net"
)

func TestClientServerNoAddrPrefix(t *testing.T) {
	port, err := cmtnet.GetFreePort()
	require.NoError(t, err)
	addr := fmt.Sprintf("localhost:%d", port)
	transport := "socket"
	app := kvstore.NewApplication()

	server, err := abciserver.NewServer(addr, transport, app)
	assert.NoError(t, err, "expected no error on NewServer")
	err = server.Start()
	assert.NoError(t, err, "expected no error on server.Start")
	defer func() { _ = server.Stop() }()

	client, err := abciclient.NewClient(addr, transport, true)
	assert.NoError(t, err, "expected no error on NewClient")
	err = client.Start()
	assert.NoError(t, err, "expected no error on client.Start")

	_ = client.Stop()
}
