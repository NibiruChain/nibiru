package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cosmossdk.io/depinject"

	wasmdapp "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/store/streaming"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/cosmos/ibc-go/v7/testing/types"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/eth"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"

	// force call init() of the geth tracers
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

const (
	appName      = "Nibiru"
	DisplayDenom = "NIBI"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = ModuleBasicManager()

	// module account permissions
	maccPerms = ModuleAccPerms()
)

var (
	_ runtime.AppI            = (*NibiruApp)(nil)
	_ servertypes.Application = (*NibiruApp)(nil)
	_ ibctesting.TestingApp   = (*NibiruApp)(nil)
)

// NibiruApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type NibiruApp struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	AppKeepers // embed all module keepers

	// the module manager
	// ModuleManager *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// module configurator
	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".nibid")
}

// GetWasmOpts build wasm options
func GetWasmOpts(
	nibiru NibiruApp,
	appOpts servertypes.AppOptions,
	wasmMsgHandlerArgs wasmext.MsgHandlerArgs,
) []wasmkeeper.Option {
	var wasmOpts []wasmkeeper.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}

	return append(wasmOpts, wasmext.NibiruWasmOptions(
		nibiru.GRPCQueryRouter(),
		nibiru.appCodec,
		wasmMsgHandlerArgs,
	)...)
}

const DefaultMaxTxGasWanted uint64 = 0

// overrideWasmVariables overrides the wasm variables to:
//   - allow for larger wasm files
func overrideWasmVariables() {
	// Override Wasm size limitation from WASMD.
	wasmtypes.MaxWasmSize = 3 * 1024 * 1024 // 3MB
	wasmtypes.MaxProposalWasmSize = wasmtypes.MaxWasmSize
}

// NewNibiruApp returns a reference to an initialized NibiruApp.
func NewNibiruApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	encodingConfig EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *NibiruApp {
	keys, _, _ := initStoreKeys()
	overrideWasmVariables()
	// baseAppOptions = append(baseAppOptions, func(app *baseapp.BaseApp) {
	// 	mp := mempool.NoOpMempool{}
	// 	app.SetMempool(mp)
	// 	handler := baseapp.NewDefaultProposalHandler(mp, app)
	// 	app.SetPrepareProposal(handler.PrepareProposalHandler())
	// 	app.SetProcessProposal(handler.ProcessProposalHandler())
	// })

	var (
		app        = &NibiruApp{}
		appBuilder *runtime.AppBuilder
		appConfig  = depinject.Configs(
			AppConfig,
			depinject.Supply(
				// supply the application options
				appOpts,

				// ADVANCED CONFIGURATION

				//
				// AUTH
				//
				// For providing a custom function required in auth to generate custom account types
				// add it below. By default the auth module uses simulation.RandomGenesisAccounts.
				//
				// authtypes.RandomGenesisAccountsFn(simulation.RandomGenesisAccounts),

				// For providing a custom a base account type add it below.
				// By default the auth module uses authtypes.ProtoBaseAccount().
				//
				func() authtypes.AccountI { return eth.ProtoBaseAccount() },

				//
				// MINT
				//

				// For providing a custom inflation function for x/mint add here your
				// custom function that implements the minttypes.InflationCalculationFn
				// interface.
			),
		)
	)

	if err := depinject.Inject(appConfig,
		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AccountKeeper,
		&app.BankKeeper,
		&app.capabilityKeeper,
		&app.StakingKeeper,
		&app.slashingKeeper,
		&app.DistrKeeper,
		&app.GovKeeper,
		&app.crisisKeeper,
		&app.upgradeKeeper,
		&app.paramsKeeper,
		&app.authzKeeper,
		&app.evidenceKeeper,
		&app.FeeGrantKeeper,
		&app.ConsensusParamsKeeper,
		// &app.DevGasKeeper,
		&app.TokenFactoryKeeper,
		&app.SudoKeeper,
		&app.OracleKeeper,
		&app.EpochsKeeper,
		&app.InflationKeeper,
		&app.EvmKeeper,
	); err != nil {
		panic(err)
	}

	nibiruBankKeeper := &evmkeeper.NibiruBankKeeper{
		BaseKeeper: app.BankKeeper.(bankkeeper.BaseKeeper),
		StateDB:    nil,
	}
	app.BankKeeper = nibiruBankKeeper

	app.App = appBuilder.Build(logger, db, traceStore, baseAppOptions...)

	app.ScopedIBCKeeper = app.capabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	app.ScopedICAControllerKeeper = app.capabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	app.ScopedICAHostKeeper = app.capabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	// scopedFeeMockKeeper := app.capabilityKeeper.ScopeToModule(MockFeePort)
	app.ScopedTransferKeeper = app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	_ = app.capabilityKeeper.ScopeToModule(ibcmock.ModuleName)

	// ibc params keepers
	app.paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	app.paramsKeeper.Subspace(ibcexported.ModuleName)
	app.paramsKeeper.Subspace(ibcfeetypes.ModuleName)
	app.paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	app.paramsKeeper.Subspace(icahosttypes.SubModuleName)
	// wasm params keepers
	app.paramsKeeper.Subspace(wasmtypes.ModuleName)

	// IBC keepers
	app.RegisterStores(keys[ibcexported.StoreKey])
	app.ibcKeeper = ibckeeper.NewKeeper(
		app.appCodec,
		keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.upgradeKeeper,
		app.ScopedIBCKeeper,
	)
	ibcModule := ibc.NewAppModule(app.ibcKeeper)

	// IBC Fee Module keeper
	app.RegisterStores(keys[ibcfeetypes.StoreKey])
	app.ibcFeeKeeper = ibcfeekeeper.NewKeeper(
		app.appCodec, keys[ibcfeetypes.StoreKey],
		app.ibcKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)
	ibcFeeModule := ibcfee.NewAppModule(app.ibcFeeKeeper)

	app.RegisterStores(keys[ibctransfertypes.StoreKey])
	app.ibcTransferKeeper = ibctransferkeeper.NewKeeper(
		app.appCodec,
		keys[ibctransfertypes.StoreKey],
		/* paramSubspace */ app.GetSubspace(ibctransfertypes.ModuleName),
		/* ibctransfertypes.ICS4Wrapper */ app.ibcFeeKeeper,
		/* ibctransfertypes.ChannelKeeper */ app.ibcKeeper.ChannelKeeper,
		/* ibctransfertypes.PortKeeper */ &app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
	)
	ibcTransferModule := ibctransfer.NewAppModule(app.ibcTransferKeeper)

	app.RegisterStores(keys[icacontrollertypes.StoreKey])
	app.icaControllerKeeper = icacontrollerkeeper.NewKeeper(
		app.appCodec, keys[icacontrollertypes.StoreKey],
		app.GetSubspace(icacontrollertypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.ScopedICAControllerKeeper,
		app.MsgServiceRouter(),
	)
	icaModule := ica.NewAppModule(&app.icaControllerKeeper, &app.icaHostKeeper)

	app.RegisterStores(keys[icahosttypes.StoreKey])
	app.icaHostKeeper = icahostkeeper.NewKeeper(
		app.appCodec,
		keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.ScopedICAHostKeeper,
		app.MsgServiceRouter(),
	)

	supportedFeatures := strings.Join(wasmdapp.AllCapabilities(), ",")

	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	wasmDir := filepath.Join(homePath, "data")
	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}
	// Create wasm VM outside keeper so it can be re-used in client keeper
	wasmVM, err := wasmvm.NewVM(filepath.Join(wasmDir, "wasm"), supportedFeatures, wasmVmContractMemoryLimit, wasmConfig.ContractDebugMode, wasmConfig.MemoryCacheSize)
	if err != nil {
		panic(err)
	}
	app.RegisterStores(keys[ibcwasmtypes.StoreKey])
	app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithVM(
		app.appCodec,
		keys[ibcwasmtypes.StoreKey],
		app.ibcKeeper.ClientKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		wasmVM,
		app.GRPCQueryRouter(),
	)
	ibcWasmModule := ibcwasm.NewAppModule(app.WasmClientKeeper)

	if err := app.RegisterModules(ibcModule, ibcFeeModule, ibcTransferModule, icaModule, ibcWasmModule); err != nil {
		panic(err)
	}

	// load state streaming if enabled
	if _, _, err := streaming.LoadStreamingServices(app.App.BaseApp, appOpts, app.appCodec, logger, keys); err != nil {
		logger.Error("failed to load state streaming", "err", err)
		os.Exit(1)
	}

	app.ModuleManager.RegisterInvariants(app.crisisKeeper)
	// RegisterUpgradeHandlers is used for registering any on-chain upgrades.
	app.setupUpgrades()
	app.sm.RegisterStoreDecoders()

	if snapshotManager := app.SnapshotManager(); snapshotManager != nil {
		if err := snapshotManager.RegisterExtensions(
			wasmkeeper.NewWasmSnapshotter(
				app.CommitMultiStore(),
				&app.WasmKeeper,
			),
			ibcwasmkeeper.NewWasmSnapshotter(
				app.CommitMultiStore(),
				&app.WasmClientKeeper,
			),
		); err != nil {
			panic("failed to add wasm snapshot extension.")
		}
	}

	if err := app.Load(loadLatest); err != nil {
		panic(err)
	}

	return app
}

// Name returns the name of the App
func (app *NibiruApp) Name() string { return app.BaseApp.Name() }

func (app *NibiruApp) RegisterNodeService(clientCtx client.Context) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *NibiruApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *NibiruApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns SimApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *NibiruApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns App's InterfaceRegistry
func (app *NibiruApp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *NibiruApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, ok := app.paramsKeeper.GetSubspace(moduleName)
	if !ok {
		panic(fmt.Errorf("failed to get subspace for module: %s", moduleName))
	}
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *NibiruApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *NibiruApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *NibiruApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(
		app.BaseApp.GRPCQueryRouter(), clientCtx,
		app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *NibiruApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// ------------------------------------------------------------------------
// Functions for ibc-go TestingApp
// ------------------------------------------------------------------------

/* GetBaseApp, GetStakingKeeper, GetIBCKeeper, and GetScopedIBCKeeper are part
   of the implementation of the TestingApp interface
*/

func (app *NibiruApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

func (app *NibiruApp) GetStakingKeeper() types.StakingKeeper {
	return app.StakingKeeper
}

func (app *NibiruApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.ibcKeeper
}

func (app *NibiruApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

/* EncodingConfig specifies the concrete encoding types to use for a given app.
   This is provided for compatibility between protobuf and amino implementations. */

func (app *NibiruApp) GetTxConfig() client.TxConfig {
	return MakeEncodingConfig().TxConfig
}

// ------------------------------------------------------------------------
// Else
// ------------------------------------------------------------------------

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(ctx client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}
