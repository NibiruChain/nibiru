package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
)

// StartJSONRPC starts the JSON-RPC server
func StartJSONRPC(
	ctx *server.Context,
	clientCtx client.Context,
	tmRPCAddr,
	tmEndpoint string,
	config *srvconfig.Config,
	indexer eth.EVMTxIndexer,
) (*http.Server, chan struct{}, error) {
	tmWsClientForRPCApi := ConnectTmWS(tmRPCAddr, tmEndpoint, ctx.Logger)

	// Configure the go-ethereum logger to sync with the ctx.Logger
	gethLogger := gethlog.NewLogger(&LogHandler{
		CmtLogger: ctx.Logger.With("module", "geth"),
	})
	gethlog.SetDefault(gethLogger)

	rpcServer := gethrpc.NewServer()

	allowUnprotectedTxs := config.JSONRPC.AllowUnprotectedTxs
	rpcAPIArr := config.JSONRPC.API

	apis := rpcapi.GetRPCAPIs(ctx, clientCtx, tmWsClientForRPCApi, allowUnprotectedTxs, indexer, rpcAPIArr)

	for _, api := range apis {
		if err := rpcServer.RegisterName(api.Namespace, api.Service); err != nil {
			gethLogger.Error(
				"failed to register service in JSON RPC namespace",
				"namespace", api.Namespace,
				"service", api.Service,
			)
			return nil, nil, err
		}
	}

	r := mux.NewRouter()
	r.HandleFunc("/", rpcServer.ServeHTTP).Methods("POST")

	handlerWithCors := cors.Default()
	if config.API.EnableUnsafeCORS {
		handlerWithCors = cors.AllowAll()
	}

	httpSrv := &http.Server{
		Addr:              config.JSONRPC.Address,
		Handler:           handlerWithCors.Handler(r),
		ReadHeaderTimeout: config.JSONRPC.HTTPTimeout,
		ReadTimeout:       config.JSONRPC.HTTPTimeout,
		WriteTimeout:      config.JSONRPC.HTTPTimeout,
		IdleTimeout:       config.JSONRPC.HTTPIdleTimeout,
	}
	httpSrvDone := make(chan struct{}, 1)

	ln, err := Listen(httpSrv.Addr, config)
	if err != nil {
		return nil, nil, err
	}

	errCh := make(chan error)
	go func() {
		ctx.Logger.Info("Starting JSON-RPC server", "address", config.JSONRPC.Address)
		if err := httpSrv.Serve(ln); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				close(httpSrvDone)
				return
			}

			ctx.Logger.Error("failed to start JSON-RPC server", "error", err.Error())
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		ctx.Logger.Error("failed to boot JSON-RPC server", "error", err.Error())
		return nil, nil, err
	case <-time.After(types.ServerStartTime): // assume JSON RPC server started successfully
	}

	ctx.Logger.Info("Starting JSON WebSocket server", "address", config.JSONRPC.WsAddress)

	// allocate separate WS connection to Tendermint
	tmWsClientForRPCWs := ConnectTmWS(tmRPCAddr, tmEndpoint, ctx.Logger)
	wsSrv := rpcapi.NewWebsocketsServer(clientCtx, ctx.Logger, tmWsClientForRPCWs, config)
	wsSrv.Start()
	return httpSrv, httpSrvDone, nil
}
