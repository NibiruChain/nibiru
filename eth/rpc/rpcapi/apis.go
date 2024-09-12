// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi/debugapi"

	rpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
)

// RPC namespaces and API version
const (
	// Cosmos namespaces
	NamespaceCosmos = "cosmos"

	// Ethereum namespaces
	NamespaceWeb3   = "web3"
	NamespaceEth    = "eth"
	NamespaceNet    = "net"
	NamespaceTxPool = "txpool"
	NamespaceDebug  = "debug"

	apiVersion = "1.0"
)

// APICreator creates the JSON-RPC API implementations.
type APICreator = func(
	ctx *server.Context,
	clientCtx client.Context,
	tendermintWebsocketClient *rpcclient.WSClient,
	allowUnprotectedTxs bool,
	indexer eth.EVMTxIndexer,
) []rpc.API

// apiCreators defines the JSON-RPC API namespaces.
var apiCreators map[string]APICreator

func init() {
	apiCreators = map[string]APICreator{
		NamespaceEth: func(ctx *server.Context,
			clientCtx client.Context,
			tmWSClient *rpcclient.WSClient,
			allowUnprotectedTxs bool,
			indexer eth.EVMTxIndexer,
		) []rpc.API {
			evmBackend := backend.NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, indexer)
			return []rpc.API{
				{
					Namespace: NamespaceEth,
					Version:   apiVersion,
					Service:   NewImplEthAPI(ctx.Logger, evmBackend),
					Public:    true,
				},
				{
					Namespace: NamespaceEth,
					Version:   apiVersion,
					Service:   NewImplFiltersAPI(ctx.Logger, clientCtx, tmWSClient, evmBackend),
					Public:    true,
				},
			}
		},
		NamespaceWeb3: func(*server.Context, client.Context, *rpcclient.WSClient, bool, eth.EVMTxIndexer) []rpc.API {
			return []rpc.API{
				{
					Namespace: NamespaceWeb3,
					Version:   apiVersion,
					Service:   NewImplWeb3API(),
					Public:    true,
				},
			}
		},
		NamespaceNet: func(_ *server.Context, clientCtx client.Context, _ *rpcclient.WSClient, _ bool, _ eth.EVMTxIndexer) []rpc.API {
			return []rpc.API{
				{
					Namespace: NamespaceNet,
					Version:   apiVersion,
					Service:   NewImplNetAPI(clientCtx),
					Public:    true,
				},
			}
		},
		NamespaceTxPool: func(ctx *server.Context, _ client.Context, _ *rpcclient.WSClient, _ bool, _ eth.EVMTxIndexer) []rpc.API {
			return []rpc.API{
				{
					Namespace: NamespaceTxPool,
					Version:   apiVersion,
					Service:   NewImplTxPoolAPI(ctx.Logger),
					Public:    true,
				},
			}
		},
		NamespaceDebug: func(ctx *server.Context,
			clientCtx client.Context,
			_ *rpcclient.WSClient,
			allowUnprotectedTxs bool,
			indexer eth.EVMTxIndexer,
		) []rpc.API {
			evmBackend := backend.NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, indexer)
			return []rpc.API{
				{
					Namespace: NamespaceDebug,
					Version:   apiVersion,
					Service:   debugapi.NewImplDebugAPI(ctx, evmBackend),
					Public:    true,
				},
			}
		},
	}
}

// GetRPCAPIs returns the list of all APIs
func GetRPCAPIs(ctx *server.Context,
	clientCtx client.Context,
	tmWSClient *rpcclient.WSClient,
	allowUnprotectedTxs bool,
	indexer eth.EVMTxIndexer,
	selectedAPIs []string,
) []rpc.API {
	var apis []rpc.API

	for _, ns := range selectedAPIs {
		if creator, ok := apiCreators[ns]; ok {
			apis = append(apis, creator(ctx, clientCtx, tmWSClient, allowUnprotectedTxs, indexer)...)
		} else {
			ctx.Logger.Error("invalid namespace value", "namespace", ns)
		}
	}

	return apis
}
