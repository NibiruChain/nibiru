package proxy

import (
	"fmt"

	abcicli "github.com/NibiruChain/nibiru/v2/bft/abci/client"
	"github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmtsync "github.com/NibiruChain/nibiru/v2/bft/libs/sync"
)

//go:generate ../scripts/mockery_generate.sh ClientCreator

// ClientCreator creates new ABCI clients.
type ClientCreator interface {
	// NewABCIClient returns a new ABCI client.
	NewABCIClient() (abcicli.Client, error)
}

//----------------------------------------------------
// local proxy uses a mutex on an in-proc app

type localClientCreator struct {
	mtx *cmtsync.Mutex
	app types.Application
}

// NewLocalClientCreator returns a ClientCreator for the given app,
// which will be running locally.
func NewLocalClientCreator(app types.Application) ClientCreator {
	return &localClientCreator{
		mtx: new(cmtsync.Mutex),
		app: app,
	}
}

func (l *localClientCreator) NewABCIClient() (abcicli.Client, error) {
	return abcicli.NewLocalClient(l.mtx, l.app), nil
}

//---------------------------------------------------------------
// remote proxy opens new connections to an external app process

type remoteClientCreator struct {
	addr        string
	transport   string
	mustConnect bool
}

// NewRemoteClientCreator returns a ClientCreator for the given address (e.g.
// "192.168.0.1") and transport (e.g. "tcp"). Set mustConnect to true if you
// want the client to connect before reporting success.
func NewRemoteClientCreator(addr, transport string, mustConnect bool) ClientCreator {
	return &remoteClientCreator{
		addr:        addr,
		transport:   transport,
		mustConnect: mustConnect,
	}
}

func (r *remoteClientCreator) NewABCIClient() (abcicli.Client, error) {
	remoteApp, err := abcicli.NewClient(r.addr, r.transport, r.mustConnect)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	return remoteApp, nil
}

// DefaultClientCreator returns a default ClientCreator. It creates a local
// client running an empty BaseApplication if addr is "noop", and a remote
// client for any other addr (for example "tcp://127.0.0.1:26658" or
// "unix:///path/to/app.sock") over the given transport ("socket" or "grpc").
//
// Nibiru BFT: upstream also accepted "kvstore", "persistent_kvstore", and
// "e2e" as built-in example apps. Those cases were removed so the example
// and e2e test apps are not linked into nibid. The standalone cometbft
// binary resolves them itself (see bft/cmd/cometbft/apps.go), and tests use
// NewLocalClientCreator with the app they need.
//
// The dbDir argument is kept for signature compatibility. It is unused.
func DefaultClientCreator(addr, transport, dbDir string) ClientCreator {
	switch addr {
	case "noop":
		return NewLocalClientCreator(types.NewBaseApplication())
	default:
		mustConnect := false // loop retrying
		return NewRemoteClientCreator(addr, transport, mustConnect)
	}
}
