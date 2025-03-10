package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"cosmossdk.io/depinject"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cmtos "github.com/cometbft/cometbft/libs/os"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/types"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"

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
	interfaceRegistry codectypes.InterfaceRegistry
	txConfig          client.TxConfig

	// keys to access the substores
	AppKeepers // embed all module keepers

	// simulation manager
	sm *module.SimulationManager
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".nibid")

	SetPrefixes("nibi")
	sdk.SetAddrCacheEnabled(false)

	blockAccAddrs = func(m map[string]bool) []string {
		k := make([]string, 0, len(m))
		for key := range m {
			k = append(k, key)
		}
		return k
	}(BlockedAddresses())
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
	overrideWasmVariables()
	baseAppOptions = append(baseAppOptions, func(app *baseapp.BaseApp) {
		mp := mempool.NoOpMempool{}
		app.SetMempool(mp)
		handler := baseapp.NewDefaultProposalHandler(mp, app)
		app.SetPrepareProposal(handler.PrepareProposalHandler())
		app.SetProcessProposal(handler.ProcessProposalHandler())
	})

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
	); err != nil {
		panic(err)
	}

	app.App = appBuilder.Build(logger, db, traceStore, baseAppOptions...)

	wasmConfig := app.InitKeepers(appOpts)

	// -------------------------- Module Options --------------------------

	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()

	app.ModuleManager.RegisterInvariants(app.crisisKeeper)

	app.EvmKeeper.AddPrecompiles(precompile.InitPrecompiles(app.AppKeepers.PublicKeepers))

	app.setupUpgrades()
	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.

	// add test gRPC service for testing gRPC queries in isolation
	testdata.RegisterQueryServer(app.GRPCQueryRouter(), testdata.QueryImpl{})

	// initialize stores
	// app.MountKVStores(keys)

	// initialize BaseApp
	anteHandler := NewAnteHandler(app.AppKeepers, ante.AnteHandlerOptions{
		HandlerOptions: authante.HandlerOptions{
			AccountKeeper:          app.AccountKeeper,
			BankKeeper:             app.BankKeeper,
			FeegrantKeeper:         app.FeeGrantKeeper,
			SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
			SigGasConsumer:         authante.DefaultSigVerificationGasConsumer,
			ExtensionOptionChecker: func(*codectypes.Any) bool { return true },
		},
		IBCKeeper:         app.ibcKeeper,
		TxCounterStoreKey: app.GetKey(wasmtypes.StoreKey),
		WasmConfig:        &wasmConfig,
		DevGasKeeper:      &app.DevGasKeeper,
		DevGasBankKeeper:  app.BankKeeper,
		// TODO: feat(evm): enable app/server/config flag for Evm MaxTxGasWanted.
		MaxTxGasWanted: DefaultMaxTxGasWanted,
		EvmKeeper:      app.EvmKeeper,
		AccountKeeper:  app.AccountKeeper,
	})

	app.SetAnteHandler(anteHandler)

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

	if loadLatest {
		if err := app.Load(loadLatest); err != nil {
			panic(err)
		}

		ctx := app.BaseApp.NewUncachedContext(true, cmtproto.Header{})

		// Initialize pinned codes in wasmvm as they are not persisted there
		if err := ibcwasmkeeper.InitializePinnedCodes(ctx, app.appCodec); err != nil {
			cmtos.Exit(fmt.Sprintf("failed to initialize pinned codes %s", err))
		}

		/* Applications that wish to enforce statically created ScopedKeepers should
		call `Seal` after creating their scoped modules in `NewApp` with
		`capabilityKeeper.ScopeToModule`.


		Calling 'app.capabilityKeeper.Seal()' initializes and seals the capability
		keeper such that all persistent capabilities are loaded in-memory and prevent
		any further modules from creating scoped sub-keepers.

		NOTE: This must be done during creation of baseapp rather than in InitChain so
		that in-memory capabilities get regenerated on app restart.
		Note that since this reads from the store, we can only perform the seal
		when `loadLatest` is set to true.
		*/
		app.capabilityKeeper.Seal()
	}

	return app
}

// Name returns the name of the App
func (app *NibiruApp) Name() string { return app.BaseApp.Name() }

// LoadHeight loads a particular height
func (app *NibiruApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

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

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	sk := app.UnsafeFindStoreKey(storeKey)
	kvStoreKey, ok := sk.(*storetypes.KVStoreKey)
	if !ok {
		return nil
	}
	return kvStoreKey
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
