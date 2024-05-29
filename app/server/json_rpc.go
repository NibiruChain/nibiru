package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	// "github.com/NibiruChain/nibiru/eth"
	// "github.com/NibiruChain/nibiru/eth/rpc/rpcapi"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	ethlog "github.com/ethereum/go-ethereum/log"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	srvconfig "github.com/NibiruChain/nibiru/app/server/config"
)

// StartJSONRPC starts the JSON-RPC server
func StartJSONRPC(ctx context.Context,
	clientCtx client.Context,
	srvCtx *server.Context,
	logger log.Logger,
	tmRPCAddr,
	tmEndpoint string,
	config *srvconfig.Config,
	// indexer eth.EVMTxIndexer,
) (*http.Server, chan struct{}, error) {
	// tmWsClient := ConnectTmWS(tmRPCAddr, tmEndpoint, logger)

	ethlog.Root().SetHandler(ethlog.FuncHandler(func(r *ethlog.Record) error {
		switch r.Lvl {
		case ethlog.LvlTrace, ethlog.LvlDebug:
			logger.Debug(r.Msg, r.Ctx...)
		case ethlog.LvlInfo, ethlog.LvlWarn:
			logger.Info(r.Msg, r.Ctx...)
		case ethlog.LvlError, ethlog.LvlCrit:
			logger.Error(r.Msg, r.Ctx...)
		}
		return nil
	}))

	rpcServer := ethrpc.NewServer()

	// allowUnprotectedTxs := config.JSONRPC.AllowUnprotectedTxs
	// rpcAPIArr := config.JSONRPC.API

	// apis := rpcapi.GetRPCAPIs(srvCtx, clientCtx, tmWsClient, allowUnprotectedTxs, indexer, rpcAPIArr)

	// for _, api := range apis {
	// 	if err := rpcServer.RegisterName(api.Namespace, api.Service); err != nil {
	// 		logger.Error(
	// 			"failed to register service in JSON RPC namespace",
	// 			"namespace", api.Namespace,
	// 			"service", api.Service,
	// 		)
	// 		return nil, nil, err
	// 	}
	// }

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
		logger.Info("Starting JSON-RPC server", "address", config.JSONRPC.Address)
		if err := httpSrv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				close(httpSrvDone)
				return
			}

			logger.Error("failed to start JSON-RPC server", "error", err.Error())
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logger.Error("failed to boot JSON-RPC server", "error", err.Error())
		return nil, nil, err

	case <-ctx.Done():
		// The calling process canceled or closed the provided context, so we must
		// gracefully stop the gRPC server.
		logger.Info("stopping gRPC server...", "address", config.GRPC.Address)

		return nil, nil, fmt.Errorf("gRPC server stopped")
	case <-time.After(ServerStartTime): // assume JSON RPC server started successfully
	}

	logger.Info("Starting JSON WebSocket server", "address", config.JSONRPC.WsAddress)

	// allocate separate WS connection to Tendermint
	// tmWsClient = ConnectTmWS(tmRPCAddr, tmEndpoint, logger)
	// wsSrv := rpcapi.NewWebsocketsServer(clientCtx, logger, tmWsClient, config)
	// wsSrv.Start()
	return httpSrv, httpSrvDone, nil
}
