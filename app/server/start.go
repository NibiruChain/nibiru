package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/indexer"

	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/cosmos/rosetta"

	"github.com/spf13/cobra"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	abciserver "github.com/cometbft/cometbft/abci/server"
	tcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	dbm "github.com/cosmos/cosmos-db"

	ethmetricsexp "github.com/ethereum/go-ethereum/metrics/exp"

	sdkioerrors "cosmossdk.io/errors"
	pruningtypes "cosmossdk.io/store/pruning/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	"github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StartOptions defines options that can be customized in `StartCmd`
type StartOptions struct {
	AppCreator      types.AppCreator
	DefaultNodeHome string
}

// NewDefaultStartOptions use the default db opener provided in tm-db.
func NewDefaultStartOptions(appCreator types.AppCreator, defaultNodeHome string) StartOptions {
	return StartOptions{
		AppCreator:      appCreator,
		DefaultNodeHome: defaultNodeHome,
	}
}

func openDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("application", backendType, dataDir)
}

// StartCmd runs the service passed in, either stand-alone or in-process with
// Tendermint.
func StartCmd(opts StartOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		Long: `Run the full node application with Tendermint in or out of process. By
default, the application will run with Tendermint in process.

Pruning options can be provided via the '--pruning' flag or alternatively with '--pruning-keep-recent',
'pruning-keep-every', and 'pruning-interval' together.

For '--pruning' the options are as follows:

default: the last 100 states are kept in addition to every 500th state; pruning at 10 block intervals
nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
everything: all saved states will be deleted, storing only the current state; pruning at 10 block intervals
custom: allow pruning options to be manually specified through 'pruning-keep-recent', 'pruning-keep-every', and 'pruning-interval'

Node halting configurations exist in the form of two flags: '--halt-height' and '--halt-time'. During
the ABCI Commit phase, the node will check if the current block height is greater than or equal to
the halt-height or if the current block time is greater than or equal to the halt-time. If so, the
node will attempt to gracefully shutdown and the block will not be committed. In addition, the node
will not be able to commit subsequent blocks.

For profiling and benchmarking purposes, CPU profiling can be enabled via the '--cpu-profile' flag
which accepts a path for the resulting pprof file.
`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := sdkserver.GetServerContextFromCmd(cmd)

			// Bind flags to the Context's Viper so the app construction can set
			// options accordingly.
			err := serverCtx.Viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			_, err = sdkserver.GetPruningOptionsFromFlags(serverCtx.Viper)
			return err
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := sdkserver.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			withTM, _ := cmd.Flags().GetBool(WithComet)
			if !withTM {
				serverCtx.Logger.Info("starting ABCI without CometBFT")
				return startStandAlone(serverCtx, opts)
			}

			serverCtx.Logger.Info("Unlocking keyring")

			// fire unlock precess for keyring
			keyringBackend, _ := cmd.Flags().GetString(flags.FlagKeyringBackend)
			if keyringBackend == keyring.BackendFile {
				_, err = clientCtx.Keyring.List()
				if err != nil {
					return err
				}
			}

			serverCtx.Logger.Info("starting ABCI with CometBFT")

			// amino is needed here for backwards compatibility of REST routes
			if err := startInProcess(serverCtx, clientCtx, opts); err != nil {
				return err
			}

			serverCtx.Logger.Debug("received quit signal")
			if err != nil {
				serverCtx.Logger.Error(fmt.Sprintf("error on quit: %s", err.Error()))
			}
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, opts.DefaultNodeHome, "The application home directory")
	cmd.Flags().Bool(WithComet, true, "Run abci app embedded in-process with CometBFT")
	cmd.Flags().String(Address, "tcp://0.0.0.0:26658", "Listen address")
	cmd.Flags().String(Transport, "socket", "Transport protocol: socket, grpc")
	cmd.Flags().String(TraceStore, "", "Enable KVStore tracing to an output file")
	cmd.Flags().String(sdkserver.FlagMinGasPrices, "", "Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 5000unibi)") //nolint:lll
	cmd.Flags().IntSlice(sdkserver.FlagUnsafeSkipUpgrades, []int{}, "Skip a set of upgrade heights to continue the old binary")
	cmd.Flags().Uint64(sdkserver.FlagHaltHeight, 0, "Block height at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Uint64(sdkserver.FlagHaltTime, 0, "Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Bool(sdkserver.FlagInterBlockCache, true, "Enable inter-block caching")
	cmd.Flags().String(CPUProfile, "", "Enable CPU profiling and write to the provided file")
	cmd.Flags().Bool(sdkserver.FlagTrace, false, "Provide full stack traces for errors in ABCI Log")
	cmd.Flags().String(sdkserver.FlagPruning, pruningtypes.PruningOptionDefault, "Pruning strategy (default|nothing|everything|custom)")
	cmd.Flags().Uint64(sdkserver.FlagPruningKeepRecent, 0, "Number of recent heights to keep on disk (ignored if pruning is not 'custom')")
	cmd.Flags().Uint64(sdkserver.FlagPruningInterval, 0, "Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')") //nolint:lll
	cmd.Flags().Uint(sdkserver.FlagInvCheckPeriod, 0, "Assert registered invariants every N blocks")
	cmd.Flags().Uint64(sdkserver.FlagMinRetainBlocks, 0, "Minimum block height offset during ABCI commit to prune Tendermint blocks")
	cmd.Flags().String(AppDBBackend, "", "The type of database for application and snapshots databases")

	cmd.Flags().Bool(GRPCOnly, false, "Start the node in gRPC query only mode without Tendermint process")
	cmd.Flags().Bool(GRPCEnable, config.DefaultGRPCEnable, "Define if the gRPC server should be enabled")
	cmd.Flags().String(GRPCAddress, serverconfig.DefaultGRPCAddress, "the gRPC server address to listen on")
	cmd.Flags().Bool(GRPCWebEnable, config.DefaultGRPCWebEnable, "Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled.)")

	cmd.Flags().Bool(RPCEnable, config.DefaultAPIEnable, "Defines if Cosmos-sdk REST server should be enabled")
	cmd.Flags().Bool(EnabledUnsafeCors, false, "Defines if CORS should be enabled (unsafe - use it at your own risk)")

	cmd.Flags().Bool(JSONRPCEnable, config.DefaultJSONRPCEnable, "Define if the JSON-RPC server should be enabled")
	cmd.Flags().StringSlice(JSONRPCAPI, config.GetDefaultAPINamespaces(), "Defines a list of JSON-RPC namespaces that should be enabled")
	cmd.Flags().String(JSONRPCAddress, config.DefaultJSONRPCAddress, "the JSON-RPC server address to listen on")
	cmd.Flags().String(JSONWsAddress, config.DefaultJSONRPCWsAddress, "the JSON-RPC WS server address to listen on")
	cmd.Flags().Uint64(JSONRPCGasCap, config.DefaultEthCallGasLimit, "Sets a cap on gas that can be used in eth_call/estimateGas unit is unibi (0=infinite)") //nolint:lll
	cmd.Flags().Float64(JSONRPCTxFeeCap, config.DefaultTxFeeCap, "Sets a cap on transaction fee that can be sent via the RPC APIs (1 = default 1 nibi)")      //nolint:lll
	cmd.Flags().Int32(JSONRPCFilterCap, config.DefaultFilterCap, "Sets the global cap for total number of filters that can be created")
	cmd.Flags().Duration(JSONRPCEVMTimeout, config.DefaultEVMTimeout, "Sets a timeout used for eth_call (0=infinite)")
	cmd.Flags().Duration(JSONRPCHTTPTimeout, config.DefaultHTTPTimeout, "Sets a read/write timeout for json-rpc http server (0=infinite)")
	cmd.Flags().Duration(JSONRPCHTTPIdleTimeout, config.DefaultHTTPIdleTimeout, "Sets a idle timeout for json-rpc http server (0=infinite)")
	cmd.Flags().Bool(JSONRPCAllowUnprotectedTxs, config.DefaultAllowUnprotectedTxs, "Allow for unprotected (non EIP155 signed) transactions to be submitted via the node's RPC when the global parameter is disabled") //nolint:lll
	cmd.Flags().Int32(JSONRPCLogsCap, config.DefaultLogsCap, "Sets the max number of results can be returned from single `eth_getLogs` query")
	cmd.Flags().Int32(JSONRPCBlockRangeCap, config.DefaultBlockRangeCap, "Sets the max block range allowed for `eth_getLogs` query")
	cmd.Flags().Int(JSONRPCMaxOpenConnections, config.DefaultMaxOpenConnections, "Sets the maximum number of simultaneous connections for the server listener") //nolint:lll
	cmd.Flags().Bool(JSONRPCEnableIndexer, false, "Enable the custom tx indexer for json-rpc")
	cmd.Flags().Bool(JSONRPCEnableMetrics, false, "Define if EVM rpc metrics server should be enabled")

	cmd.Flags().String(EVMTracer, config.DefaultEVMTracer, "the EVM tracer type to collect execution traces from the EVM transaction execution (json|struct|access_list|markdown)") //nolint:lll
	cmd.Flags().Uint64(EVMMaxTxGasWanted, config.DefaultMaxTxGasWanted, "the gas wanted for each eth tx returned in ante handler in check tx mode")                                 //nolint:lll

	cmd.Flags().String(TLSCertPath, "", "the cert.pem file path for the server TLS configuration")
	cmd.Flags().String(TLSKeyPath, "", "the key.pem file path for the server TLS configuration")

	cmd.Flags().Uint64(sdkserver.FlagStateSyncSnapshotInterval, 0, "State sync snapshot interval")
	cmd.Flags().Uint32(sdkserver.FlagStateSyncSnapshotKeepRecent, 2, "State sync snapshot to keep")

	// add support for all Tendermint-specific command line options
	tcmd.AddNodeFlags(cmd)
	return cmd
}

func startStandAlone(svrCtx *sdkserver.Context, opts StartOptions) error {
	addr := svrCtx.Viper.GetString(Address)
	transport := svrCtx.Viper.GetString(Transport)
	home := svrCtx.Viper.GetString(flags.FlagHome)

	db, err := openDB(home, sdkserver.GetAppDBBackend(svrCtx.Viper))
	if err != nil {
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			svrCtx.Logger.Error("error closing db", "error", err.Error())
		}
	}()

	traceWriterFile := svrCtx.Viper.GetString(TraceStore)
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		return err
	}

	app := opts.AppCreator(svrCtx.Logger, db, traceWriter, svrCtx.Viper)

	conf, err := config.GetConfig(svrCtx.Viper)
	if err != nil {
		svrCtx.Logger.Error("failed to get server config", "error", err.Error())
		return err
	}

	if err := conf.ValidateBasic(); err != nil {
		svrCtx.Logger.Error("invalid server config", "error", err.Error())
		return err
	}

	_, err = startTelemetry(conf)
	if err != nil {
		return err
	}

	cmtApp := sdkserver.NewCometABCIWrapper(app)
	svr, err := abciserver.NewServer(addr, transport, cmtApp)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}

	svr.SetLogger(servercmtlog.CometLoggerWrapper{Logger: svrCtx.Logger.With("server", "abci")})
	g, ctx := getCtx(svrCtx, false)

	g.Go(func() error {
		if err := svr.Start(); err != nil {
			svrCtx.Logger.Error("failed to start out-of-process ABCI server", "err", err)
			return err
		}

		// Wait for the calling process to be canceled or close the provided context,
		// so we can gracefully stop the ABCI server.
		<-ctx.Done()
		svrCtx.Logger.Info("stopping the ABCI server...")
		return svr.Stop()
	})

	return g.Wait()
}

// legacyAminoCdc is used for the legacy REST API
func startInProcess(svrCtx *sdkserver.Context, clientCtx client.Context, opts StartOptions) (err error) {
	cfg := svrCtx.Config
	home := cfg.RootDir
	logger := svrCtx.Logger
	g, ctx := getCtx(svrCtx, true)

	db, err := openDB(home, sdkserver.GetAppDBBackend(svrCtx.Viper))
	if err != nil {
		logger.Error("failed to open DB", "error", err.Error())
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			svrCtx.Logger.With("error", err).Error("error closing db")
		}
	}()

	traceWriterFile := svrCtx.Viper.GetString(TraceStore)
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		logger.Error("failed to open trace writer", "error", err.Error())
		return err
	}

	conf, err := config.GetConfig(svrCtx.Viper)
	if err != nil {
		logger.Error("failed to get server config", "error", err.Error())
		return err
	}

	if err := conf.ValidateBasic(); err != nil {
		logger.Error("invalid server config", "error", err.Error())
		return err
	}

	app := opts.AppCreator(svrCtx.Logger, db, traceWriter, svrCtx.Viper)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		logger.Error("failed load or gen node key", "error", err.Error())
		return err
	}

	genDocProvider := node.DefaultGenesisDocProviderFunc(cfg)

	var (
		tmNode   *node.Node
		gRPCOnly = svrCtx.Viper.GetBool(GRPCOnly)
	)

	if gRPCOnly {
		logger.Info("starting node in query only mode; CometBFT is disabled")
		conf.GRPC.Enable = true
		conf.JSONRPC.EnableIndexer = false
	} else {
		logger.Info("starting node with ABCI CometBFT in-process")

		cmtApp := sdkserver.NewCometABCIWrapper(app)
		tmNode, err = node.NewNode(
			cfg,
			pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
			nodeKey,
			proxy.NewLocalClientCreator(cmtApp),
			genDocProvider,
			cmtcfg.DefaultDBProvider,
			node.DefaultMetricsProvider(cfg.Instrumentation),
			servercmtlog.CometLoggerWrapper{Logger: svrCtx.Logger.With("server", "node")},
		)
		if err != nil {
			logger.Error("failed init node", "error", err.Error())
			return err
		}

		if err := tmNode.Start(); err != nil {
			logger.Error("failed start tendermint server", "error", err.Error())
			return err
		}

		defer func() {
			if tmNode.IsRunning() {
				_ = tmNode.Stop()
			}
		}()
	}

	// Add the tx service to the gRPC router. We only need to register this
	// service if API or gRPC or JSONRPC is enabled, and avoid doing so in the general
	// case, because it spawns a new local tendermint RPC rpcClient.
	if (conf.API.Enable || conf.GRPC.Enable || conf.JSONRPC.Enable || conf.JSONRPC.EnableIndexer) && tmNode != nil {
		clientCtx = clientCtx.WithClient(local.New(tmNode))

		app.RegisterTxService(clientCtx)
		app.RegisterTendermintService(clientCtx)
		app.RegisterNodeService(clientCtx, conf.Config)
	}

	metrics, err := startTelemetry(conf)
	if err != nil {
		return err
	}

	// Enable metrics if JSONRPC is enabled and --metrics is passed
	// Flag not added in config to avoid user enabling in config without passing in CLI
	if conf.JSONRPC.Enable && svrCtx.Viper.GetBool(JSONRPCEnableMetrics) {
		ethmetricsexp.Setup(conf.JSONRPC.MetricsAddress)
	}

	var evmIdxer eth.EVMTxIndexer
	if conf.JSONRPC.EnableIndexer {
		idxDB, err := OpenIndexerDB(home, sdkserver.GetAppDBBackend(svrCtx.Viper))
		if err != nil {
			logger.Error("failed to open evm indexer DB", "error", err.Error())
			return err
		}
		evmTxIndexer, evmIndexerService, err := OpenEVMIndexer(svrCtx, idxDB, clientCtx)
		if err != nil {
			logger.Error("failed starting evm indexer service", "error", err.Error())
			return err
		}
		evmIdxer = evmTxIndexer
		defer func() {
			if err := evmIndexerService.Stop(); err != nil {
				svrCtx.Logger.Error("failed to stop evm indexer service", "error", err.Error())
			}
		}()
	}

	if conf.API.Enable || conf.JSONRPC.Enable {
		genDoc, err := genDocProvider()
		if err != nil {
			return err
		}

		clientCtx = clientCtx.
			WithHomeDir(home).
			WithChainID(genDoc.ChainID)
	}

	grpcSrv, clientCtx, err := startGrpcServer(ctx, svrCtx, clientCtx, g, conf.GRPC, app)
	if err != nil {
		return err
	}
	if grpcSrv != nil {
		defer grpcSrv.GracefulStop()
	}

	apiSrv := startAPIServer(ctx, svrCtx, clientCtx, g, conf.Config, app, grpcSrv, metrics)

	if apiSrv != nil {
		defer apiSrv.Close()
	}

	clientCtx, httpSrv, httpSrvDone, err := startJSONRPCServer(svrCtx, clientCtx, g, conf, genDocProvider, cfg.RPC.ListenAddress, evmIdxer)

	if httpSrv != nil {
		defer func() {
			shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFn()
			if err := httpSrv.Shutdown(shutdownCtx); err != nil {
				logger.Error("HTTP server shutdown produced a warning", "error", err.Error())
			} else {
				logger.Info("HTTP server shut down, waiting 5 sec")
				select {
				case <-time.Tick(5 * time.Second):
				case <-httpSrvDone:
				}
			}
		}()
	}

	// At this point it is safe to block the process if we're in query only mode as
	// we do not need to start Rosetta or handle any Tendermint related processes.
	if gRPCOnly {
		// wait for signal capture and gracefully return
		return g.Wait()
	}

	if err := startRosettaServer(svrCtx, clientCtx, g, conf); err != nil {
		return err
	}

	// wait for signal capture and gracefully return
	// we are guaranteed to be waiting for the "ListenForQuitSignals" goroutine.
	return g.Wait()
}

// OpenIndexerDB opens the custom eth indexer db, using the same db backend as the main app
func OpenIndexerDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("evmindexer", backendType, dataDir)
}

func OpenEVMIndexer(
	ctx *sdkserver.Context, indexerDb dbm.DB, clientCtx client.Context,
) (eth.EVMTxIndexer, *EVMTxIndexerService, error) {
	idxLogger := ctx.Logger.With("indexer", "evm")
	evmIndexer := indexer.NewEVMTxIndexer(indexerDb, idxLogger, clientCtx)

	evmIndexerService := NewEVMIndexerService(evmIndexer, clientCtx.Client.(cmtrpcclient.Client))
	evmIndexerService.SetLogger(servercmtlog.CometLoggerWrapper{Logger: idxLogger})

	errCh := make(chan error)
	go func() {
		if err := evmIndexerService.Start(); err != nil {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return nil, nil, err
	case <-time.After(ServerStartTime): // assume server started successfully
	}
	return evmIndexer, evmIndexerService, nil
}

func openTraceWriter(traceWriterFile string) (w io.Writer, err error) {
	if traceWriterFile == "" {
		return
	}

	filePath := filepath.Clean(traceWriterFile)
	return os.OpenFile(
		filePath,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0o600,
	)
}

func startTelemetry(cfg config.Config) (*telemetry.Metrics, error) {
	if !cfg.Telemetry.Enabled {
		return nil, nil
	}
	return telemetry.New(cfg.Telemetry)
}

func getCtx(svrCtx *sdkserver.Context, block bool) (*errgroup.Group, context.Context) {
	ctx, cancelFn := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	// listen for quit signals so the calling parent process can gracefully exit
	sdkserver.ListenForQuitSignals(g, block, cancelFn, svrCtx.Logger)
	return g, ctx
}

func startGrpcServer(
	ctx context.Context,
	svrCtx *sdkserver.Context,
	clientCtx client.Context,
	g *errgroup.Group,
	config serverconfig.GRPCConfig,
	app types.Application,
) (*grpc.Server, client.Context, error) {
	if !config.Enable {
		// return grpcServer as nil if gRPC is disabled
		return nil, clientCtx, nil
	}
	_, port, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, clientCtx, sdkioerrors.Wrapf(err, "invalid grpc address %s", config.Address)
	}

	maxSendMsgSize := config.MaxSendMsgSize
	if maxSendMsgSize == 0 {
		maxSendMsgSize = serverconfig.DefaultGRPCMaxSendMsgSize
	}

	maxRecvMsgSize := config.MaxRecvMsgSize
	if maxRecvMsgSize == 0 {
		maxRecvMsgSize = serverconfig.DefaultGRPCMaxRecvMsgSize
	}

	grpcAddress := fmt.Sprintf("127.0.0.1:%s", port)

	// if gRPC is enabled, configure gRPC client for gRPC gateway and json-rpc
	grpcClient, err := grpc.NewClient(
		grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodec(codec.NewProtoCodec(clientCtx.InterfaceRegistry).GRPCCodec()),
			grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
			grpc.MaxCallSendMsgSize(maxSendMsgSize),
		),
	)
	if err != nil {
		return nil, clientCtx, err
	}
	// Set `GRPCClient` to `clientCtx` to enjoy concurrent grpc query.
	// only use it if gRPC server is enabled.
	clientCtx = clientCtx.WithGRPCClient(grpcClient)
	svrCtx.Logger.Debug("gRPC client assigned to client context", "address", config.Address)

	grpcSrv, err := servergrpc.NewGRPCServer(clientCtx, app, config)
	if err != nil {
		return nil, clientCtx, err
	}

	// Start the gRPC server in a goroutine. Note, the provided ctx will ensure
	// that the server is gracefully shut down.
	g.Go(func() error {
		return servergrpc.StartGRPCServer(ctx, svrCtx.Logger.With("module", "grpc-server"), config, grpcSrv)
	})
	return grpcSrv, clientCtx, nil
}

func startAPIServer(
	ctx context.Context,
	svrCtx *sdkserver.Context,
	clientCtx client.Context,
	g *errgroup.Group,
	svrCfg serverconfig.Config,
	app types.Application,
	grpcSrv *grpc.Server,
	metrics *telemetry.Metrics,
) *api.Server {
	if !svrCfg.API.Enable {
		return nil
	}

	apiSrv := api.New(clientCtx, svrCtx.Logger.With("server", "api"), grpcSrv)
	app.RegisterAPIRoutes(apiSrv, svrCfg.API)

	if svrCfg.Telemetry.Enabled {
		apiSrv.SetTelemetry(metrics)
	}

	g.Go(func() error {
		return apiSrv.Start(ctx, svrCfg)
	})
	return apiSrv
}

func startJSONRPCServer(
	svrCtx *sdkserver.Context,
	clientCtx client.Context,
	g *errgroup.Group,
	config config.Config,
	genDocProvider node.GenesisDocProvider,
	cmtRPCAddr string,
	idxer eth.EVMTxIndexer,
) (ctx client.Context, httpSrv *http.Server, httpSrvDone chan struct{}, err error) {
	ctx = clientCtx
	if !config.JSONRPC.Enable {
		return
	}

	genDoc, err := genDocProvider()
	if err != nil {
		return ctx, httpSrv, httpSrvDone, err
	}

	ctx = clientCtx.WithChainID(genDoc.ChainID)
	cmtEndpoint := "/websocket"
	g.Go(func() error {
		httpSrv, httpSrvDone, err = StartJSONRPC(svrCtx, clientCtx, cmtRPCAddr, cmtEndpoint, &config, idxer)
		return err
	})
	return
}

func startRosettaServer(
	svrCtx *sdkserver.Context,
	clientCtx client.Context,
	g *errgroup.Group,
	config config.Config,
) error {
	if !config.Rosetta.Enable {
		return nil
	}

	offlineMode := config.Rosetta.Offline

	// If GRPC is not enabled rosetta cannot work in online mode, so it works in
	// offline mode.
	if !config.GRPC.Enable {
		offlineMode = true
	}

	minGasPrices, err := sdk.ParseDecCoins(config.MinGasPrices)
	if err != nil {
		svrCtx.Logger.Error("failed to parse minimum-gas-prices", "error", err.Error())
		return err
	}

	conf := &rosetta.Config{
		Blockchain:          config.Rosetta.Blockchain,
		Network:             config.Rosetta.Network,
		TendermintRPC:       svrCtx.Config.RPC.ListenAddress,
		GRPCEndpoint:        config.GRPC.Address,
		Addr:                config.Rosetta.Addr,
		Retries:             config.Rosetta.Retries,
		Offline:             offlineMode,
		GasToSuggest:        config.Rosetta.GasToSuggest,
		EnableFeeSuggestion: config.Rosetta.EnableFeeSuggestion,
		GasPrices:           minGasPrices.Sort(),
		Codec:               clientCtx.Codec.(*codec.ProtoCodec),
		InterfaceRegistry:   clientCtx.InterfaceRegistry,
	}

	rosettaSrv, err := rosetta.ServerFromConfig(conf)
	if err != nil {
		return err
	}

	g.Go(rosettaSrv.Start)
	return nil
}
