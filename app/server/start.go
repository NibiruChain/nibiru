package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/indexer"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	dbm "github.com/cometbft/cometbft-db"
	abciserver "github.com/cometbft/cometbft/abci/server"
	tcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"

	"cosmossdk.io/tools/rosetta"
	crgserver "cosmossdk.io/tools/rosetta/lib/server"

	ethmetricsexp "github.com/ethereum/go-ethereum/metrics/exp"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	"github.com/cosmos/cosmos-sdk/server/types"
	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"
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

			withTM, _ := cmd.Flags().GetBool(WithTendermint)
			if !withTM {
				serverCtx.Logger.Info("starting ABCI without Tendermint")
				return wrapCPUProfile(serverCtx, func() error {
					return startStandAlone(serverCtx, opts)
				})
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

			serverCtx.Logger.Info("starting ABCI with Tendermint")

			// amino is needed here for backwards compatibility of REST routes
			err = startInProcess(serverCtx, clientCtx, opts)
			errCode, ok := err.(sdkserver.ErrorCode)
			if !ok {
				return err
			}

			serverCtx.Logger.Debug(fmt.Sprintf("received quit signal: %d", errCode.Code))
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, opts.DefaultNodeHome, "The application home directory")
	cmd.Flags().Bool(WithTendermint, true, "Run abci app embedded in-process with tendermint")
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
	cmd.Flags().String(GRPCWebAddress, serverconfig.DefaultGRPCWebAddress, "The gRPC-Web server address to listen on")

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

func startStandAlone(ctx *sdkserver.Context, opts StartOptions) error {
	addr := ctx.Viper.GetString(Address)
	transport := ctx.Viper.GetString(Transport)
	home := ctx.Viper.GetString(flags.FlagHome)

	db, err := openDB(home, sdkserver.GetAppDBBackend(ctx.Viper))
	if err != nil {
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			ctx.Logger.Error("error closing db", "error", err.Error())
		}
	}()

	traceWriterFile := ctx.Viper.GetString(TraceStore)
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		return err
	}

	app := opts.AppCreator(ctx.Logger, db, traceWriter, ctx.Viper)

	conf, err := config.GetConfig(ctx.Viper)
	if err != nil {
		ctx.Logger.Error("failed to get server config", "error", err.Error())
		return err
	}

	if err := conf.ValidateBasic(); err != nil {
		ctx.Logger.Error("invalid server config", "error", err.Error())
		return err
	}

	_, err = startTelemetry(conf)
	if err != nil {
		return err
	}

	svr, err := abciserver.NewServer(addr, transport, app)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}

	svr.SetLogger(ctx.Logger.With("server", "abci"))

	err = svr.Start()
	if err != nil {
		tmos.Exit(err.Error())
	}

	defer func() {
		if err = svr.Stop(); err != nil {
			tmos.Exit(err.Error())
		}
	}()

	// Wait for SIGINT or SIGTERM signal
	return sdkserver.WaitForQuitSignals()
}

// legacyAminoCdc is used for the legacy REST API
func startInProcess(ctx *sdkserver.Context, clientCtx client.Context, opts StartOptions) (err error) {
	cfg := ctx.Config
	home := cfg.RootDir
	logger := ctx.Logger

	db, err := openDB(home, sdkserver.GetAppDBBackend(ctx.Viper))
	if err != nil {
		logger.Error("failed to open DB", "error", err.Error())
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			ctx.Logger.With("error", err).Error("error closing db")
		}
	}()

	traceWriterFile := ctx.Viper.GetString(TraceStore)
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		logger.Error("failed to open trace writer", "error", err.Error())
		return err
	}

	conf, err := config.GetConfig(ctx.Viper)
	if err != nil {
		logger.Error("failed to get server config", "error", err.Error())
		return err
	}

	if err := conf.ValidateBasic(); err != nil {
		logger.Error("invalid server config", "error", err.Error())
		return err
	}

	app := opts.AppCreator(ctx.Logger, db, traceWriter, ctx.Viper)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		logger.Error("failed load or gen node key", "error", err.Error())
		return err
	}

	genDocProvider := node.DefaultGenesisDocProviderFunc(cfg)

	var (
		tmNode   *node.Node
		gRPCOnly = ctx.Viper.GetBool(GRPCOnly)
	)

	if gRPCOnly {
		logger.Info("starting node in query only mode; Tendermint is disabled")
		conf.GRPC.Enable = true
		conf.JSONRPC.EnableIndexer = false
	} else {
		logger.Info("starting node with ABCI Tendermint in-process")

		tmNode, err = node.NewNode(
			cfg,
			pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
			nodeKey,
			proxy.NewLocalClientCreator(app),
			genDocProvider,
			node.DefaultDBProvider,
			node.DefaultMetricsProvider(cfg.Instrumentation),
			ctx.Logger.With("server", "node"),
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
		app.RegisterNodeService(clientCtx)
	}

	metrics, err := startTelemetry(conf)
	if err != nil {
		return err
	}

	// Enable metrics if JSONRPC is enabled and --metrics is passed
	// Flag not added in config to avoid user enabling in config without passing in CLI
	if conf.JSONRPC.Enable && ctx.Viper.GetBool(JSONRPCEnableMetrics) {
		ethmetricsexp.Setup(conf.JSONRPC.MetricsAddress)
	}

	var evmIdxer eth.EVMTxIndexer
	if conf.JSONRPC.EnableIndexer {
		idxDB, err := OpenIndexerDB(home, sdkserver.GetAppDBBackend(ctx.Viper))
		if err != nil {
			logger.Error("failed to open evm indexer DB", "error", err.Error())
			return err
		}
		evmTxIndexer, _, err := OpenEVMIndexer(ctx, idxDB, clientCtx)
		if err != nil {
			logger.Error("failed starting evm indexer service", "error", err.Error())
			return err
		}
		evmIdxer = evmTxIndexer
	}

	if conf.API.Enable || conf.JSONRPC.Enable {
		genDoc, err := genDocProvider()
		if err != nil {
			return err
		}

		clientCtx = clientCtx.
			WithHomeDir(home).
			WithChainID(genDoc.ChainID)

		// Set `GRPCClient` to `clientCtx` to enjoy concurrent grpc query.
		// only use it if gRPC server is enabled.
		if conf.GRPC.Enable {
			_, port, err := net.SplitHostPort(conf.GRPC.Address)
			if err != nil {
				return errorsmod.Wrapf(err, "invalid grpc address %s", conf.GRPC.Address)
			}

			maxSendMsgSize := conf.GRPC.MaxSendMsgSize
			if maxSendMsgSize == 0 {
				maxSendMsgSize = serverconfig.DefaultGRPCMaxSendMsgSize
			}

			maxRecvMsgSize := conf.GRPC.MaxRecvMsgSize
			if maxRecvMsgSize == 0 {
				maxRecvMsgSize = serverconfig.DefaultGRPCMaxRecvMsgSize
			}

			grpcAddress := fmt.Sprintf("127.0.0.1:%s", port)

			// If grpc is enabled, configure grpc rpcClient for grpc gateway and json-rpc.
			grpcClient, err := grpc.Dial(
				grpcAddress,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithDefaultCallOptions(
					grpc.ForceCodec(codec.NewProtoCodec(clientCtx.InterfaceRegistry).GRPCCodec()),
					grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
					grpc.MaxCallSendMsgSize(maxSendMsgSize),
				),
			)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.WithGRPCClient(grpcClient)
			ctx.Logger.Debug("gRPC rpcClient assigned to rpcClient context", "address", grpcAddress)
		}
	}

	var apiSrv *api.Server
	if conf.API.Enable {
		apiSrv = api.New(clientCtx, ctx.Logger.With("server", "api"))
		app.RegisterAPIRoutes(apiSrv, conf.API)

		if conf.Telemetry.Enabled {
			apiSrv.SetTelemetry(metrics)
		}

		errCh := make(chan error)
		go func() {
			if err := apiSrv.Start(conf.Config); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err
		case <-time.After(types.ServerStartTime): // assume server started successfully
		}

		defer apiSrv.Close()
	}

	var (
		grpcSrv    *grpc.Server
		grpcWebSrv *http.Server
	)

	if conf.GRPC.Enable {
		grpcSrv, err = servergrpc.StartGRPCServer(clientCtx, app, conf.GRPC)
		if err != nil {
			return err
		}
		defer grpcSrv.Stop()
		if conf.GRPCWeb.Enable {
			grpcWebSrv, err = servergrpc.StartGRPCWeb(grpcSrv, conf.Config)
			if err != nil {
				ctx.Logger.Error("failed to start grpc-web http server", "error", err.Error())
				return err
			}

			defer func() {
				if err := grpcWebSrv.Close(); err != nil {
					logger.Error("failed to close the grpc-web http server", "error", err.Error())
				}
			}()
		}
	}

	var (
		httpSrv     *http.Server
		httpSrvDone chan struct{}
	)

	if conf.JSONRPC.Enable {
		genDoc, err := genDocProvider()
		if err != nil {
			return err
		}

		clientCtx := clientCtx.WithChainID(genDoc.ChainID)

		tmEndpoint := "/websocket"
		tmRPCAddr := cfg.RPC.ListenAddress
		httpSrv, httpSrvDone, err = StartJSONRPC(
			ctx, clientCtx, tmRPCAddr, tmEndpoint, &conf, evmIdxer,
		)
		if err != nil {
			return err
		}
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
		return sdkserver.WaitForQuitSignals()
	}

	var rosettaSrv crgserver.Server
	if conf.Rosetta.Enable {
		offlineMode := conf.Rosetta.Offline

		// If GRPC is not enabled rosetta cannot work in online mode, so it works in
		// offline mode.
		if !conf.GRPC.Enable {
			offlineMode = true
		}

		minGasPrices, err := sdk.ParseDecCoins(conf.MinGasPrices)
		if err != nil {
			ctx.Logger.Error("failed to parse minimum-gas-prices", "error", err.Error())
			return err
		}

		conf := &rosetta.Config{
			Blockchain:          conf.Rosetta.Blockchain,
			Network:             conf.Rosetta.Network,
			TendermintRPC:       ctx.Config.RPC.ListenAddress,
			GRPCEndpoint:        conf.GRPC.Address,
			Addr:                conf.Rosetta.Address,
			Retries:             conf.Rosetta.Retries,
			Offline:             offlineMode,
			GasToSuggest:        conf.Rosetta.GasToSuggest,
			EnableFeeSuggestion: conf.Rosetta.EnableFeeSuggestion,
			GasPrices:           minGasPrices.Sort(),
			Codec:               clientCtx.Codec.(*codec.ProtoCodec),
			InterfaceRegistry:   clientCtx.InterfaceRegistry,
		}

		rosettaSrv, err = rosetta.ServerFromConfig(conf)
		if err != nil {
			return err
		}

		errCh := make(chan error)
		go func() {
			if err := rosettaSrv.Start(); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err
		case <-time.After(types.ServerStartTime): // assume server started successfully
		}
	}
	// Wait for SIGINT or SIGTERM signal
	return sdkserver.WaitForQuitSignals()
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

	evmIndexerService := NewEVMIndexerService(evmIndexer, clientCtx.Client.(rpcclient.Client))
	evmIndexerService.SetLogger(idxLogger)

	errCh := make(chan error)
	go func() {
		if err := evmIndexerService.Start(); err != nil {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return nil, nil, err
	case <-time.After(types.ServerStartTime): // assume server started successfully
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

// WaitForQuitSignals waits for SIGINT and SIGTERM and returns.
func WaitForQuitSignals() sdkserver.ErrorCode {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	return sdkserver.ErrorCode{Code: int(sig.(syscall.Signal)) + 128}
}

// wrapCPUProfile runs callback in a goroutine, then wait for quit signals.
func wrapCPUProfile(ctx *sdkserver.Context, callback func() error) error {
	if cpuProfile := ctx.Viper.GetString(CPUProfile); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return err
		}

		ctx.Logger.Info("starting CPU profiler", "profile", cpuProfile)
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		defer func() {
			ctx.Logger.Info("stopping CPU profiler", "profile", cpuProfile)
			pprof.StopCPUProfile()
			if err := f.Close(); err != nil {
				ctx.Logger.Info("failed to close cpu-profile file", "profile", cpuProfile, "err", err.Error())
			}
		}()
	}

	errCh := make(chan error)
	go func() {
		errCh <- callback()
	}()

	select {
	case err := <-errCh:
		return err

	case <-time.After(types.ServerStartTime):
	}

	return WaitForQuitSignals()
}
