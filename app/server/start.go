package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"

	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/hashicorp/go-metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"golang.org/x/sync/errgroup"

	"github.com/NibiruChain/nibiru/app/server/config"

	// "github.com/NibiruChain/nibiru/eth/indexer"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server/api"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/spf13/cobra"

	abciserver "github.com/cometbft/cometbft/abci/server"
	tcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	"github.com/cometbft/cometbft/rpc/client/local"
	dbm "github.com/cosmos/cosmos-db"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"

	pruningtypes "cosmossdk.io/store/pruning/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/server/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

const ServerStartTime = 5 * time.Second

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
// CometBFT.
func StartCmd(appCreator types.AppCreator, defaultNodeHome string) *cobra.Command {
	return StartCmdWithOptions(appCreator, defaultNodeHome, server.StartCmdOptions{})
}

// StartCmd runs the service passed in, either stand-alone or in-process with
// Tendermint.
func StartCmdWithOptions(appCreator types.AppCreator, defaultNodeHome string, opts server.StartCmdOptions) *cobra.Command {
	if opts.DBOpener == nil {
		opts.DBOpener = openDB
	}

	if opts.StartCommandHandler == nil {
		opts.StartCommandHandler = start
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		Long: `Run the full node application with CometBFT in or out of process. By
default, the application will run with CometBFT in process.

Pruning options can be provided via the '--pruning' flag or alternatively with '--pruning-keep-recent',
'pruning-keep-every', and 'pruning-interval' together.

For '--pruning' the options are as follows:

default: the last 362880 states are kept, pruning at 10 block intervals
nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
everything: 2 latest states will be kept; pruning at 10 block intervals.
custom: allow pruning options to be manually specified through 'pruning-keep-recent', 'pruning-keep-every', and 'pruning-interval'

Node halting configurations exist in the form of two flags: '--halt-height' and '--halt-time'. During
the ABCI Commit phase, the node will check if the current block height is greater than or equal to
the halt-height or if the current block time is greater than or equal to the halt-time. If so, the
node will attempt to gracefully shutdown and the block will not be committed. In addition, the node
will not be able to commit subsequent blocks.

For profiling and benchmarking purposes, CPU profiling can be enabled via the '--cpu-profile' flag
which accepts a path for the resulting pprof file.

The node may be started in a 'query only' mode where only the gRPC and JSON HTTP
API services are enabled via the 'grpc-only' flag. In this mode, CometBFT is
bypassed and can be used when legacy queries are needed after an on-chain upgrade
is performed. Note, when enabled, gRPC will also be automatically enabled.
`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)

			// Bind flags to the Context's Viper so the app construction can set
			// options accordingly.
			err := serverCtx.Viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			_, err = server.GetPruningOptionsFromFlags(serverCtx.Viper)
			return err
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			withCMT, _ := cmd.Flags().GetBool(WithTendermint)
			if !withCMT {
				serverCtx.Logger.Info("starting ABCI without Tendermint")
			}

			serverCtx.Logger.Info("starting ABCI with Tendermint")
			err = wrapCPUProfile(serverCtx, func() error {
				return opts.StartCommandHandler(serverCtx, clientCtx, appCreator, withCMT, opts)
			})

			if err != nil {
				serverCtx.Logger.Error("failed to start ABCI with Tendermint", "error", err.Error())
				return err
			}

			serverCtx.Logger.Debug("received quit signal")
			graceDuration, _ := cmd.Flags().GetDuration(server.FlagShutdownGrace)
			if graceDuration > 0 {
				serverCtx.Logger.Info("graceful shutdown start", server.FlagShutdownGrace, graceDuration)
				<-time.After(graceDuration)
				serverCtx.Logger.Info("graceful shutdown complete")
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

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	addStartNodeFlags(cmd, opts)
	return cmd
}

func start(svrCtx *server.Context, clientCtx client.Context, appCreator types.AppCreator, withCmt bool, opts server.StartCmdOptions) error {
	svrCfg, err := getAndValidateConfig(svrCtx)
	if err != nil {
		return err
	}

	app, appCleanupFn, err := startApp(svrCtx, appCreator, opts)
	if err != nil {
		return err
	}
	defer appCleanupFn()

	metrics, err := startTelemetry(svrCfg)
	if err != nil {
		return err
	}

	emitServerInfoMetrics()

	if !withCmt {
		return startStandAlone(svrCtx, svrCfg, clientCtx, app, metrics, opts)
	}
	return startInProcess(svrCtx, svrCfg, clientCtx, app, metrics, opts)
}

func startStandAlone(svrCtx *server.Context, svrCfg config.Config, clientCtx client.Context, app types.Application, metrics *telemetry.Metrics, opts server.StartCmdOptions) error {
	addr := svrCtx.Viper.GetString(Address)
	transport := svrCtx.Viper.GetString(Transport)

	cmtApp := server.NewCometABCIWrapper(app)
	svr, err := abciserver.NewServer(addr, transport, cmtApp)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}

	svr.SetLogger(servercmtlog.CometLoggerWrapper{Logger: svrCtx.Logger.With("module", "abci-server")})

	g, ctx := getCtx(svrCtx, false)

	// Add the tx service to the gRPC router. We only need to register this
	// service if API or gRPC is enabled, and avoid doing so in the general
	// case, because it spawns a new local CometBFT RPC client.
	if svrCfg.API.Enable || svrCfg.GRPC.Enable {
		// create tendermint client
		// assumes the rpc listen address is where tendermint has its rpc server
		rpcclient, err := rpchttp.New(svrCtx.Config.RPC.ListenAddress, "/websocket")
		if err != nil {
			return err
		}
		// re-assign for making the client available below
		// do not use := to avoid shadowing clientCtx
		clientCtx = clientCtx.WithClient(rpcclient)

		// use the provided clientCtx to register the services
		app.RegisterTxService(clientCtx)
		app.RegisterTendermintService(clientCtx)
		app.RegisterNodeService(clientCtx, svrCfg.Config)
	}

	grpcSrv, clientCtx, err := startGrpcServer(ctx, g, svrCfg.GRPC, clientCtx, svrCtx, app)
	if err != nil {
		return err
	}

	err = startAPIServer(ctx, g, svrCfg.Config, clientCtx, svrCtx, app, svrCtx.Config.RootDir, grpcSrv, metrics)
	if err != nil {
		return err
	}

	if opts.PostSetupStandalone != nil {
		if err := opts.PostSetupStandalone(svrCtx, clientCtx, ctx, g); err != nil {
			return err
		}
	}

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
func startInProcess(svrCtx *server.Context, svrCfg config.Config, clientCtx client.Context, app types.Application,
	metrics *telemetry.Metrics, opts server.StartCmdOptions,
) error {
	cfg := svrCtx.Config

	gRPCOnly := svrCtx.Viper.GetBool(GRPCOnly)

	g, ctx := getCtx(svrCtx, true)

	if gRPCOnly {
		// TODO: Generalize logic so that gRPC only is really in startStandAlone
		svrCtx.Logger.Info("starting node in gRPC only mode; CometBFT is disabled")
		svrCfg.GRPC.Enable = true
	} else {
		svrCtx.Logger.Info("starting node with ABCI CometBFT in-process")
		tmNode, cleanupFn, err := startCmtNode(ctx, cfg, app, svrCtx)
		if err != nil {
			return err
		}
		defer cleanupFn()

		// Add the tx service to the gRPC router. We only need to register this
		// service if API or gRPC is enabled, and avoid doing so in the general
		// case, because it spawns a new local CometBFT RPC client.
		if svrCfg.API.Enable || svrCfg.GRPC.Enable {
			// Re-assign for making the client available below do not use := to avoid
			// shadowing the clientCtx variable.
			clientCtx = clientCtx.WithClient(local.New(tmNode))

			app.RegisterTxService(clientCtx)
			app.RegisterTendermintService(clientCtx)
			app.RegisterNodeService(clientCtx, svrCfg.Config)
		}
	}

	err := startJsonServer(ctx, clientCtx, svrCfg, svrCtx)
	if err != nil {
		return err
	}

	grpcSrv, clientCtx, err := startGrpcServer(ctx, g, svrCfg.GRPC, clientCtx, svrCtx, app)
	if err != nil {
		return err
	}

	err = startAPIServer(ctx, g, svrCfg.Config, clientCtx, svrCtx, app, cfg.RootDir, grpcSrv, metrics)
	if err != nil {
		return err
	}

	if opts.PostSetup != nil {
		if err := opts.PostSetup(svrCtx, clientCtx, ctx, g); err != nil {
			return err
		}
	}
	// wait for signal capture and gracefully return
	// we are guaranteed to be waiting for the "ListenForQuitSignals" goroutine.
	return g.Wait()
}

type cometABCIWrapper struct {
	app types.ABCI
}

func startApp(svrCtx *server.Context, appCreator types.AppCreator, opts server.StartCmdOptions) (app types.Application, cleanupFn func(), err error) {
	traceWriter, traceCleanupFn, err := setupTraceWriter(svrCtx)
	if err != nil {
		return app, traceCleanupFn, err
	}

	home := svrCtx.Config.RootDir
	db, err := opts.DBOpener(home, server.GetAppDBBackend(svrCtx.Viper))
	if err != nil {
		return app, traceCleanupFn, err
	}

	app = appCreator(svrCtx.Logger, db, traceWriter, svrCtx.Viper)

	cleanupFn = func() {
		traceCleanupFn()
		if localErr := app.Close(); localErr != nil {
			svrCtx.Logger.Error(localErr.Error())
		}
	}
	return app, cleanupFn, nil
}

func setupTraceWriter(svrCtx *server.Context) (traceWriter io.WriteCloser, cleanup func(), err error) {
	// clean up the traceWriter when the server is shutting down
	cleanup = func() {}

	traceWriterFile := svrCtx.Viper.GetString(TraceStore)
	traceWriter, err = openTraceWriter(traceWriterFile)
	if err != nil {
		return traceWriter, cleanup, err
	}

	// if flagTraceStore is not used then traceWriter is nil
	if traceWriter != nil {
		cleanup = func() {
			if err = traceWriter.Close(); err != nil {
				svrCtx.Logger.Error("failed to close trace writer", "err", err)
			}
		}
	}

	return traceWriter, cleanup, nil
}

func startJsonServer(ctx context.Context, clientCtx client.Context, cfg config.Config, svrCtx *server.Context) error {
	logger := svrCtx.Logger
	// var idxer eth.EVMTxIndexer

	// if cfg.JSONRPC.EnableIndexer {
	// home := svrCtx.Config.RootDir
	// 	idxDB, err := OpenIndexerDB(home, server.GetAppDBBackend(svrCtx.Viper))
	// 	if err != nil {
	// 		svrCtx.Logger.Error("failed to open evm indexer DB", "error", err.Error())
	// 		return err
	// 	}

	// 	idxLogger := logger.With("indexer", "evm")
	// 	idxer = indexer.NewKVIndexer(idxDB, idxLogger, clientCtx)
	// }

	if cfg.JSONRPC.Enable {
		genDoc, err := genutiltypes.AppGenesisFromFile(svrCtx.Config.GenesisFile())
		if err != nil {
			return err
		}

		clientCtx := clientCtx.WithChainID(hackChainID(genDoc.ChainID))

		tmEndpoint := "/websocket"
		tmRPCAddr := svrCtx.Config.RPC.ListenAddress
		httpSrv, httpSrvDone, err := StartJSONRPC(ctx, clientCtx, svrCtx, logger, tmRPCAddr, tmEndpoint, &cfg)
		if err != nil {
			return err
		}
		defer func() {
			shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFn()
			if err := httpSrv.Shutdown(shutdownCtx); err != nil {
				svrCtx.Logger.Error("HTTP server shutdown produced a warning", "error", err.Error())
			} else {
				svrCtx.Logger.Info("HTTP server shut down, waiting 5 sec")
				select {
				case <-time.Tick(5 * time.Second):
				case <-httpSrvDone:
				}
			}
		}()
	}
	return nil
}

// OpenIndexerDB opens the custom eth indexer db, using the same db backend as the main app
func OpenIndexerDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("evmindexer", backendType, dataDir)
}

func openTraceWriter(traceWriterFile string) (w io.WriteCloser, err error) {
	if traceWriterFile == "" {
		return
	}
	return os.OpenFile(
		traceWriterFile,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0o666,
	)
}

func startTelemetry(cfg config.Config) (*telemetry.Metrics, error) {
	if !cfg.Telemetry.Enabled {
		return nil, nil
	}
	return telemetry.New(cfg.Telemetry)
}

// wrapCPUProfile starts CPU profiling, if enabled, and executes the provided
// callbackFn in a separate goroutine, then will wait for that callback to
// return.
//
// NOTE: We expect the caller to handle graceful shutdown and signal handling.
func wrapCPUProfile(svrCtx *server.Context, callbackFn func() error) error {
	if cpuProfile := svrCtx.Viper.GetString(CPUProfile); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return err
		}

		svrCtx.Logger.Info("starting CPU profiler", "profile", cpuProfile)

		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		defer func() {
			svrCtx.Logger.Info("stopping CPU profiler", "profile", cpuProfile)
			pprof.StopCPUProfile()

			if err := f.Close(); err != nil {
				svrCtx.Logger.Info("failed to close cpu-profile file", "profile", cpuProfile, "err", err.Error())
			}
		}()
	}

	return callbackFn()
}

// hackChainID replaces nibiru-localnet-0 with nibirulocalnet-9000-1 which matches the standard
func hackChainID(chainID string) string {
	re := regexp.MustCompile(`-\d+$`)
	lastNumber := re.FindString(chainID)
	trimmedInput := strings.TrimSuffix(chainID, lastNumber)
	if lastNumber == "-0" {
		lastNumber = "-1"
	}
	trimmedInput = strings.ReplaceAll(trimmedInput, "-", "")
	result := trimmedInput + "_9000" + lastNumber

	return result
}

// emitServerInfoMetrics emits server info related metrics using application telemetry.
func emitServerInfoMetrics() {
	var ls []metrics.Label

	versionInfo := version.NewInfo()
	if len(versionInfo.GoVersion) > 0 {
		ls = append(ls, telemetry.NewLabel("go", versionInfo.GoVersion))
	}
	if len(versionInfo.CosmosSdkVersion) > 0 {
		ls = append(ls, telemetry.NewLabel("version", versionInfo.CosmosSdkVersion))
	}

	if len(ls) == 0 {
		return
	}

	telemetry.SetGaugeWithLabels([]string{"server", "info"}, 1, ls)
}

func startGrpcServer(ctx context.Context, g *errgroup.Group, config serverconfig.GRPCConfig,
	clientCtx client.Context,
	svrCtx *server.Context,
	app types.Application) (*grpc.Server, client.Context, error) {
	if !config.Enable {
		return nil, clientCtx, nil
	}

	_, _, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, clientCtx, err
	}

	maxSendMsgSize := config.MaxSendMsgSize
	if maxSendMsgSize == 0 {
		maxSendMsgSize = serverconfig.DefaultGRPCMaxSendMsgSize
	}

	maxRecvMsgSize := config.MaxRecvMsgSize
	if maxRecvMsgSize == 0 {
		maxRecvMsgSize = serverconfig.DefaultGRPCMaxRecvMsgSize
	}

	// if gRPC is enabled, configure gRPC client for gRPC gateway
	grpcClient, err := grpc.Dial(
		config.Address,
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

	clientCtx = clientCtx.WithGRPCClient(grpcClient)
	svrCtx.Logger.Debug("gRPC client assigned to client context", "target", config.Address)

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

// returns a function which returns the genesis doc from the genesis file.
func getGenDocProvider(cfg *cmtcfg.Config) func() (*cmttypes.GenesisDoc, error) {
	return func() (*cmttypes.GenesisDoc, error) {
		appGenesis, err := genutiltypes.AppGenesisFromFile(cfg.GenesisFile())
		if err != nil {
			return nil, err
		}

		return appGenesis.ToGenesisDoc()
	}
}

func startAPIServer(
	ctx context.Context,
	g *errgroup.Group,
	svrCfg serverconfig.Config,
	clientCtx client.Context,
	svrCtx *server.Context,
	app types.Application,
	home string,
	grpcSrv *grpc.Server,
	metrics *telemetry.Metrics,
) error {
	if !svrCfg.API.Enable {
		return nil
	}

	clientCtx = clientCtx.WithHomeDir(home)

	apiSrv := api.New(clientCtx, svrCtx.Logger.With("module", "api-server"), grpcSrv)
	app.RegisterAPIRoutes(apiSrv, svrCfg.API)

	if svrCfg.Telemetry.Enabled {
		apiSrv.SetTelemetry(metrics)
	}

	g.Go(func() error {
		return apiSrv.Start(ctx, svrCfg)
	})
	return nil
}

// TODO: Move nodeKey into being created within the function.
func startCmtNode(
	ctx context.Context,
	cfg *cmtcfg.Config,
	app types.Application,
	svrCtx *server.Context,
) (tmNode *node.Node, cleanupFn func(), err error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return nil, cleanupFn, err
	}

	cmtApp := server.NewCometABCIWrapper(app)
	tmNode, err = node.NewNodeWithContext(
		ctx,
		cfg,
		pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(cmtApp),
		getGenDocProvider(cfg),
		cmtcfg.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		servercmtlog.CometLoggerWrapper{Logger: svrCtx.Logger},
	)
	if err != nil {
		return tmNode, cleanupFn, err
	}

	if err := tmNode.Start(); err != nil {
		return tmNode, cleanupFn, err
	}

	cleanupFn = func() {
		if tmNode != nil && tmNode.IsRunning() {
			_ = tmNode.Stop()
		}
	}

	return tmNode, cleanupFn, nil
}

func addStartNodeFlags(cmd *cobra.Command, opts server.StartCmdOptions) {
	cmd.Flags().Bool(WithTendermint, true, "Run abci app embedded in-process with tendermint")
	cmd.Flags().String(Address, "tcp://0.0.0.0:26658", "Listen address")
	cmd.Flags().String(Transport, "socket", "Transport protocol: socket, grpc")
	cmd.Flags().String(TraceStore, "", "Enable KVStore tracing to an output file")
	cmd.Flags().String(server.FlagMinGasPrices, "", "Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 5000unibi)") //nolint:lll
	cmd.Flags().IntSlice(server.FlagUnsafeSkipUpgrades, []int{}, "Skip a set of upgrade heights to continue the old binary")
	cmd.Flags().Uint64(server.FlagHaltHeight, 0, "Block height at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Uint64(server.FlagHaltTime, 0, "Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Bool(server.FlagInterBlockCache, true, "Enable inter-block caching")
	cmd.Flags().String(CPUProfile, "", "Enable CPU profiling and write to the provided file")
	cmd.Flags().Bool(server.FlagTrace, false, "Provide full stack traces for errors in ABCI Log")
	cmd.Flags().String(server.FlagPruning, pruningtypes.PruningOptionDefault, "Pruning strategy (default|nothing|everything|custom)")
	cmd.Flags().Uint64(server.FlagPruningKeepRecent, 0, "Number of recent heights to keep on disk (ignored if pruning is not 'custom')")
	cmd.Flags().Uint64(server.FlagPruningInterval, 0, "Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')") //nolint:lll
	cmd.Flags().Uint(server.FlagInvCheckPeriod, 0, "Assert registered invariants every N blocks")
	cmd.Flags().Uint64(server.FlagMinRetainBlocks, 0, "Minimum block height offset during ABCI commit to prune Tendermint blocks")
	cmd.Flags().String(AppDBBackend, "", "The type of database for application and snapshots databases")

	cmd.Flags().Bool(GRPCOnly, false, "Start the node in gRPC query only mode without Tendermint process")
	cmd.Flags().Bool(GRPCEnable, config.DefaultGRPCEnable, "Define if the gRPC server should be enabled")
	cmd.Flags().String(GRPCAddress, serverconfig.DefaultGRPCAddress, "the gRPC server address to listen on")
	cmd.Flags().Bool(GRPCWebEnable, config.DefaultGRPCWebEnable, "Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled.)")
	cmd.Flags().String(GRPCWebAddress, serverconfig.DefaultGRPCAddress, "The gRPC-Web server address to listen on")

	cmd.Flags().Bool(RPCEnable, config.DefaultAPIEnable, "Defines if Cosmos-sdk REST server should be enabled")
	cmd.Flags().Bool(EnabledUnsafeCors, false, "Defines if CORS should be enabled (unsafe - use it at your own risk)")

	cmd.Flags().Bool(JSONRPCEnable, config.DefaultJSONRPCEnable, "Define if the JSON-RPC server should be enabled")
	cmd.Flags().StringSlice(JSONRPCAPI, config.GetDefaultAPINamespaces(), "Defines a list of JSON-RPC namespaces that should be enabled")
	cmd.Flags().String(JSONRPCAddress, config.DefaultJSONRPCAddress, "the JSON-RPC server address to listen on")
	cmd.Flags().String(JSONWsAddress, config.DefaultJSONRPCWsAddress, "the JSON-RPC WS server address to listen on")
	cmd.Flags().Uint64(JSONRPCGasCap, config.DefaultGasCap, "Sets a cap on gas that can be used in eth_call/estimateGas unit is unibi (0=infinite)")     //nolint:lll
	cmd.Flags().Float64(JSONRPCTxFeeCap, config.DefaultTxFeeCap, "Sets a cap on transaction fee that can be sent via the RPC APIs (1 = default 1 nibi)") //nolint:lll
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

	cmd.Flags().Uint64(server.FlagStateSyncSnapshotInterval, 0, "State sync snapshot interval")
	cmd.Flags().Uint32(server.FlagStateSyncSnapshotKeepRecent, 2, "State sync snapshot to keep")

	// add support for all Tendermint-specific command line options
	tcmd.AddNodeFlags(cmd)

	if opts.AddFlags != nil {
		opts.AddFlags(cmd)
	}
}

func getCtx(svrCtx *server.Context, block bool) (*errgroup.Group, context.Context) {
	ctx, cancelFn := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	// listen for quit signals so the calling parent process can gracefully exit
	server.ListenForQuitSignals(g, block, cancelFn, svrCtx.Logger)
	return g, ctx
}

func getAndValidateConfig(svrCtx *server.Context) (config.Config, error) {
	config, err := config.GetConfig(svrCtx.Viper)
	if err != nil {
		return config, err
	}

	if err := config.ValidateBasic(); err != nil {
		return config, err
	}
	return config, nil
}
