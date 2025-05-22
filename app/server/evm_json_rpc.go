package server

import (
	"errors"
	"html/template"
	"net/http"
	"time"

	// The `_ "embed"` import adds access to files embedded in the running Go
	// program (smart contracts).
	_ "embed"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
)

//go:embed evm_json_rpc_get.html
var htmlTemplateEvmJsonRpc []byte

// StartEthereumJSONRPC starts the Ethereum JSON-RPC server and websocket server
// for Nibiru.
func StartEthereumJSONRPC(
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

	// This router for the Ethereum JSON-RPC matches on both the path ("/")
	// and method ("POST", "GET", "PUT") to choose a handler. This allows us
	// to add different behavior based on the type of request to display a
	// webpage if someone visits the RPC URL.
	r := mux.NewRouter()
	r.HandleFunc("/", rpcServer.ServeHTTP).Methods("POST")
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Vary", "User-Agent")
		w.WriteHeader(http.StatusOK)

		tmpl := template.Must(
			template.New("evm_json_rpc_get").
				Parse(string(htmlTemplateEvmJsonRpc)),
		)
		err := tmpl.Execute(w,
			struct {
				Status            string
				NowTime           string
				Web3ClientVersion string
			}{
				Status:            "Active",
				NowTime:           startTime.Format(time.DateTime),
				Web3ClientVersion: appconst.RuntimeVersion(),
			})
		if err != nil {
			http.Error(w, "Internal template error", http.StatusInternalServerError)
		}
	}).Methods("GET")

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
