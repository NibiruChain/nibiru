package main

import (
	"fmt"

	"github.com/NibiruChain/nibiru/v2/bft/abci/example/kvstore"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cfg "github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	nm "github.com/NibiruChain/nibiru/v2/bft/node"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	"github.com/NibiruChain/nibiru/v2/bft/privval"
	"github.com/NibiruChain/nibiru/v2/bft/proxy"
	e2e "github.com/NibiruChain/nibiru/v2/bft/test/e2e/app"
)

// builtinApp returns the in-process example app named by proxyApp, or false
// if proxyApp is not one of "kvstore", "persistent_kvstore", or "e2e".
//
// Nibiru BFT moved these cases out of proxy.DefaultClientCreator so that the
// example and e2e test apps are only linked into this standalone binary, not
// into nibid.
func builtinApp(proxyApp, dbDir string) (abci.Application, bool, error) {
	switch proxyApp {
	case "kvstore":
		return kvstore.NewApplication(), true, nil
	case "persistent_kvstore":
		return kvstore.NewPersistentKVStoreApplication(dbDir), true, nil
	case "e2e":
		app, err := e2e.NewApplication(e2e.DefaultConfig(dbDir))
		if err != nil {
			return nil, true, fmt.Errorf("creating e2e app: %w", err)
		}
		return app, true, nil
	default:
		return nil, false, nil
	}
}

// clientCreator resolves the built-in example apps, then falls back to
// proxy.DefaultClientCreator for "noop" and remote addresses.
func clientCreator(config *cfg.Config) (proxy.ClientCreator, error) {
	app, ok, err := builtinApp(config.ProxyApp, config.DBDir())
	if err != nil {
		return nil, err
	}
	if ok {
		return proxy.NewLocalClientCreator(app), nil
	}
	return proxy.DefaultClientCreator(
		config.ProxyApp, config.ABCI, config.DBDir(),
	), nil
}

// newNode is node.DefaultNewNode with the built-in example apps restored,
// matching upstream `cometbft node --proxy_app kvstore` behavior.
func newNode(config *cfg.Config, logger log.Logger) (*nm.Node, error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load or gen node key %s: %w", config.NodeKeyFile(), err,
		)
	}

	creator, err := clientCreator(config)
	if err != nil {
		return nil, err
	}

	return nm.NewNode(config,
		privval.LoadOrGenFilePV(
			config.PrivValidatorKeyFile(), config.PrivValidatorStateFile(),
		),
		nodeKey,
		creator,
		nm.DefaultGenesisDocProviderFunc(config),
		nm.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)
}
