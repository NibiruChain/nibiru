// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"context"
	"fmt"
	"reflect"
	"unicode"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/eth"

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
			evmBackend := NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, indexer)
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
			evmBackend := NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, indexer)
			return []rpc.API{
				{
					Namespace: NamespaceDebug,
					Version:   apiVersion,
					Service:   NewImplDebugAPI(ctx, evmBackend),
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

var (
	ctxType          = reflect.TypeOf((*context.Context)(nil)).Elem()
	errType          = reflect.TypeOf((*error)(nil)).Elem()
	subscriptionType = reflect.TypeOf(Subscription{})
)

type MethodInfo struct {
	TrueName       string
	GoFunc         reflect.Method
	IsSubscription bool
}

// ParseAPIMethods returns all method names exposed by api, e.g. "eth_gasPrice"
func ParseAPIMethods(api rpc.API) map[string]MethodInfo {
	svcType := reflect.TypeOf(api.Service)
	methods := make(map[string]MethodInfo)

	for i := range svcType.NumMethod() {
		m := svcType.Method(i)
		if m.PkgPath != "" {
			continue // unexported
		}
		sig := m.Type

		// 1) Detect optional context.Context arg
		// Args must be: Optional([context.Context]) + zero or more args
		hasCtx := sig.NumIn() > 1 && sig.In(1) == ctxType

		// 2) Validate outputs: either (error), or (T, error)
		nOut := sig.NumOut()
		switch nOut {
		case 1:
			if !sig.Out(0).Implements(errType) {
				continue
			}
		case 2:
			if !sig.Out(1).Implements(errType) {
				continue
			}
		default:
			continue
		}

		// 3) detect subscriptions: ctx + (Subscription, error)
		isSub := false
		if hasCtx && nOut == 2 {
			t0 := sig.Out(0)
			// strip pointer
			for t0.Kind() == reflect.Ptr {
				t0 = t0.Elem()
			}
			if t0 == subscriptionType {
				isSub = true
			}
		}

		// 3) name the RPC method by lower‑casing the first rune
		trueName := rpcMethodName(api.Namespace, m.Name)
		methods[trueName] = MethodInfo{
			TrueName:       trueName,
			GoFunc:         m,
			IsSubscription: isSub,
		}
	}

	return methods
}

// Example: "TransactionByHash" -> "transactionByHash"
func rpcMethodName(namespace, funcName string) string {
	return fmt.Sprintf("%v_%v", namespace, lowerFirst(funcName))
}

// lowerFirst returns lowercases the first letter of the input. For context,
// service methods are in the form:
// fmt.Sprintf("%v_%v", namespace, lowerFirst(methodName))
func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) > 0 {
		r[0] = unicode.ToLower(r[0])
	}
	return string(r)
}
