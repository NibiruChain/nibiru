package testnetwork

import (
	"fmt"
	"os"
	"time"

	"cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/NibiruChain/nibiru/v2/app/server"
	ethrpc "github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"

	"github.com/cosmos/cosmos-sdk/server/api"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	srvtypes "github.com/cosmos/cosmos-sdk/server/types"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
)

func startNodeAndServers(cfg Config, val *Validator) error {
	logger := val.Ctx.Logger
	evmServerCtxLogger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	tmCfg := val.Ctx.Config
	tmCfg.Instrumentation.Prometheus = false

	if err := val.AppConfig.ValidateBasic(); err != nil {
		return err
	}

	nodeKey, err := p2p.LoadOrGenNodeKey(tmCfg.NodeKeyFile())
	if err != nil {
		return err
	}

	app := cfg.AppConstructor(*val)

	genDocProvider := node.DefaultGenesisDocProviderFunc(tmCfg)
	tmNode, err := node.NewNode(
		tmCfg,
		pvm.LoadOrGenFilePV(tmCfg.PrivValidatorKeyFile(), tmCfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(app),
		genDocProvider,
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(tmCfg.Instrumentation),
		logger.With("module", val.Moniker),
	)
	if err != nil {
		return fmt.Errorf("failed to construct Node: %w", err)
	}

	if err := tmNode.Start(); err != nil {
		return fmt.Errorf("failed Node.Start(): %w", err)
	}

	val.tmNode = tmNode
	val.tmNode.Logger = logger

	if val.RPCAddress != "" {
		val.RPCClient = local.New(tmNode)
	}

	// We'll need a RPC client if the validator exposes a gRPC or REST endpoint.
	if val.APIAddress != "" || val.AppConfig.GRPC.Enable {
		val.ClientCtx = val.ClientCtx.
			WithClient(val.RPCClient)

		// Add the tx service in the gRPC router.
		app.RegisterTxService(val.ClientCtx)

		// Add the tendermint queries service in the gRPC router.
		app.RegisterTendermintService(val.ClientCtx)

		val.EthRpc_NET = rpcapi.NewImplNetAPI(val.ClientCtx)
	}

	if val.APIAddress != "" {
		apiSrv := api.New(val.ClientCtx, logger.With("module", "api-server"))
		app.RegisterAPIRoutes(apiSrv, val.AppConfig.API)

		errCh := make(chan error)

		go func() {
			if err := apiSrv.Start(val.AppConfig.Config); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err
		case <-time.After(srvtypes.ServerStartTime): // assume server started successfully
		}

		val.api = apiSrv
	}

	if val.AppConfig.GRPC.Enable {
		grpcSrv, err := servergrpc.StartGRPCServer(val.ClientCtx, app, val.AppConfig.GRPC)
		if err != nil {
			return err
		}

		val.grpc = grpcSrv

		if val.AppConfig.GRPCWeb.Enable {
			val.grpcWeb, err = servergrpc.StartGRPCWeb(grpcSrv, val.AppConfig.Config)
			if err != nil {
				return err
			}
		}
	}

	val.Ctx.Logger = evmServerCtxLogger

	useEthJsonRPC := val.AppConfig.JSONRPC.Enable && val.AppConfig.JSONRPC.Address != ""
	if useEthJsonRPC {
		if val.Ctx == nil || val.Ctx.Viper == nil {
			return fmt.Errorf("validator %s context is nil", val.Moniker)
		}

		tmEndpoint := "/websocket"
		tmRPCAddr := fmt.Sprintf("tcp://%s", val.AppConfig.GRPC.Address)

		val.Logger.Log("Set EVM indexer")

		homeDir := val.Ctx.Config.RootDir
		evmTxIndexer, err := server.OpenEVMIndexer(
			val.Ctx, evmServerCtxLogger, val.ClientCtx, homeDir,
		)
		if err != nil {
			return err
		}
		val.EthTxIndexer = evmTxIndexer

		val.jsonrpc, val.jsonrpcDone, err = server.StartJSONRPC(val.Ctx, val.ClientCtx, tmRPCAddr, tmEndpoint, val.AppConfig, nil)
		if err != nil {
			return errors.Wrap(err, "failed to start JSON-RPC server")
		}

		address := fmt.Sprintf("http://%s", val.AppConfig.JSONRPC.Address)

		val.JSONRPCClient, err = ethclient.Dial(address)
		if err != nil {
			return fmt.Errorf("failed to dial JSON-RPC at address %s: %w", val.AppConfig.JSONRPC.Address, err)
		}

		val.Logger.Log("Set up Ethereum JSON-RPC client objects")
		val.EthRpcQueryClient = ethrpc.NewQueryClient(val.ClientCtx)
		val.EthRpcBackend = backend.NewBackend(
			val.Ctx,
			val.Ctx.Logger,
			val.ClientCtx,
			val.AppConfig.JSONRPC.AllowUnprotectedTxs,
			val.EthTxIndexer,
		)

		val.Logger.Log("Expose typed methods for each namespace")
		val.EthRPC_ETH = rpcapi.NewImplEthAPI(val.Ctx.Logger, val.EthRpcBackend)
		val.EthRpc_WEB3 = rpcapi.NewImplWeb3API()

		val.Ctx.Logger = logger // set back to normal setting
	}

	return nil
}
