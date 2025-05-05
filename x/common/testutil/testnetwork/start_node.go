package testnetwork

import (
	"context"
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	db "github.com/cosmos/cosmos-db"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/NibiruChain/nibiru/v2/app/server"
	ethrpc "github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"

	sdkserver "github.com/cosmos/cosmos-sdk/server"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
)

func startNodeAndServers(cfg Config, val *Validator) error {
	logger := val.Ctx.Logger
	evmServerCtxLogger := sdkserver.NewDefaultContext()
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
	cmtApp := sdkserver.NewCometABCIWrapper(app)

	genDocProvider := node.DefaultGenesisDocProviderFunc(tmCfg)
	tmNode, err := node.NewNode(
		tmCfg,
		pvm.LoadOrGenFilePV(tmCfg.PrivValidatorKeyFile(), tmCfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(cmtApp),
		genDocProvider,
		cmtcfg.DefaultDBProvider,
		node.DefaultMetricsProvider(tmCfg.Instrumentation),
		servercmtlog.CometLoggerWrapper{Logger: logger.With("module", val.Moniker)},
	)
	if err != nil {
		return fmt.Errorf("failed to construct Node: %w", err)
	}

	if err := tmNode.Start(); err != nil {
		return fmt.Errorf("failed Node.Start(): %w", err)
	}

	val.tmNode = tmNode
	val.tmNode.Logger = servercmtlog.CometLoggerWrapper{Logger: logger}

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

	// We'll need a RPC client if the validator exposes a gRPC or REST endpoint.
	if val.APIAddress != "" || val.AppConfig.GRPC.Enable {
		val.ClientCtx = val.ClientCtx.
			WithClient(val.RPCClient)

		app.RegisterTxService(val.ClientCtx)
		app.RegisterTendermintService(val.ClientCtx)
		app.RegisterNodeService(val.ClientCtx, val.AppConfig.Config)
	}

	if val.AppConfig.GRPC.Enable {
		grpcSrv, err := servergrpc.NewGRPCServer(val.ClientCtx, app, val.AppConfig.GRPC)
		if err != nil {
			return err
		}

		err = servergrpc.StartGRPCServer(context.Background(), logger.With(log.ModuleKey, "grpc-server"), val.AppConfig.GRPC, grpcSrv)
		if err != nil {
			return err
		}

		val.grpc = grpcSrv
	}

	val.Ctx.Logger = evmServerCtxLogger.Logger

	useEthJsonRPC := val.AppConfig.JSONRPC.Enable && val.AppConfig.JSONRPC.Address != ""
	if useEthJsonRPC {
		if val.Ctx == nil || val.Ctx.Viper == nil {
			return fmt.Errorf("validator %s context is nil", val.Moniker)
		}

		tmEndpoint := "/websocket"
		tmRPCAddr := fmt.Sprintf("tcp://%s", val.AppConfig.GRPC.Address)

		val.Logger.Log("Set EVM indexer")

		evmTxIndexer, evmTxIndexerService, err := server.OpenEVMIndexer(val.Ctx, db.NewMemDB(), val.ClientCtx)
		if err != nil {
			{
				return fmt.Errorf("failed starting evm indexer service: %w", err)
			}
		}
		val.EthTxIndexer = evmTxIndexer
		val.EthTxIndexerService = evmTxIndexerService

		val.jsonrpc, val.jsonrpcDone, err = server.StartJSONRPC(val.Ctx, val.ClientCtx, tmRPCAddr, tmEndpoint, val.AppConfig, val.EthTxIndexer)
		if err != nil {
			return sdkioerrors.Wrap(err, "failed to start JSON-RPC server")
		}

		endpointEvmJsonRpc := fmt.Sprintf("http://%s", val.AppConfig.JSONRPC.Address)
		val.EvmRpcClient, err = ethclient.Dial(endpointEvmJsonRpc)
		if err != nil {
			return fmt.Errorf("failed to dial JSON-RPC at address %s: %w", val.AppConfig.JSONRPC.Address, err)
		}

		val.Logger.Log("Set up Ethereum JSON-RPC client objects")
		val.EthRpcQueryClient = ethrpc.NewQueryClient(val.ClientCtx)
		val.EthRpcBackend = rpcapi.NewBackend(
			val.Ctx,
			val.Ctx.Logger,
			val.ClientCtx,
			val.AppConfig.JSONRPC.AllowUnprotectedTxs,
			val.EthTxIndexer,
		)

		val.Logger.Log("Expose typed methods for each namespace")
		val.EthRPC_ETH = rpcapi.NewImplEthAPI(val.Ctx.Logger, val.EthRpcBackend)

		val.Ctx.Logger = logger // set back to normal setting
	}

	return nil
}
