package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"cosmossdk.io/depinject"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
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
	"github.com/cosmos/cosmos-sdk/store/streaming"
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
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclientclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/client"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/types"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/eth"
	cryptocodec "github.com/NibiruChain/nibiru/v2/eth/crypto/codec"
	"github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/devgas/v1"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	"github.com/NibiruChain/nibiru/v2/x/epochs"
	epochstypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmmodule"
	"github.com/NibiruChain/nibiru/v2/x/genmsg"
	"github.com/NibiruChain/nibiru/v2/x/inflation"
	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
	oracle "github.com/NibiruChain/nibiru/v2/x/oracle"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory"
	tokenfactorytypes "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

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
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		BankModule{},
		capability.AppModuleBasic{},
		StakingModule{},
		distr.AppModuleBasic{},
		NewGovModuleBasic(
			paramsclient.ProposalHandler,
			upgradeclient.LegacyProposalHandler,
			upgradeclient.LegacyCancelProposalHandler,
			ibcclientclient.UpdateClientProposalHandler,
			ibcclientclient.UpgradeProposalHandler,
		),
		params.AppModuleBasic{},
		CrisisModule{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		groupmodule.AppModuleBasic{},
		// ibc 'AppModuleBasic's
		ibc.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		ica.AppModuleBasic{},
		ibcwasm.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
		// native x/
		evmmodule.AppModuleBasic{},
		oracle.AppModuleBasic{},
		epochs.AppModuleBasic{},
		inflation.AppModuleBasic{},
		sudo.AppModuleBasic{},
		wasm.AppModuleBasic{},
		devgas.AppModuleBasic{},
		tokenfactory.AppModuleBasic{},
		genmsg.AppModule{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		inflationtypes.ModuleName:      {authtypes.Minter, authtypes.Burner},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		oracletypes.ModuleName:         {},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		ibcfeetypes.ModuleName:         {},
		icatypes.ModuleName:            {},

		evm.ModuleName:                   {authtypes.Minter, authtypes.Burner},
		epochstypes.ModuleName:           {},
		sudotypes.ModuleName:             {},
		common.TreasuryPoolModuleAccount: {},
		wasmtypes.ModuleName:             {authtypes.Burner},
		tokenfactorytypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
	}
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
	// TODO(k-yang): remove once depinject is fully integrated
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	AppKeepers // embed all module keepers

	// simulation manager
	sm *module.SimulationManager
}

func init() {
	SetPrefixes("nibi")
	sdk.DefaultBondDenom = appconst.BondDenom

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".nibid")

	// Override Wasm size limitation from WASMD.
	//   - allow for larger wasm files
	wasmtypes.MaxWasmSize = 3 * 1024 * 1024 // 3MB
	wasmtypes.MaxProposalWasmSize = wasmtypes.MaxWasmSize
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

// NewNibiruApp returns a reference to an initialized NibiruApp.
func NewNibiruApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *NibiruApp {
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
		&app.StakingKeeper,
		&app.DistrKeeper,
	); err != nil {
		panic(err)
	}
	app.App = appBuilder.Build(logger, db, traceStore, baseAppOptions...)

	// init non-depinject keys
	app.keys = sdk.NewKVStoreKeys(
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		consensusparamtypes.StoreKey,
		upgradetypes.StoreKey,
		feegrant.StoreKey,
		evidencetypes.StoreKey,
		capabilitytypes.StoreKey,
		authzkeeper.StoreKey,
		crisistypes.StoreKey,

		// ibc keys
		ibctransfertypes.StoreKey,
		ibcfeetypes.StoreKey,
		ibcexported.StoreKey,
		icahosttypes.StoreKey,
		icacontrollertypes.StoreKey,
		ibcwasmtypes.StoreKey,

		// nibiru x/ keys
		oracletypes.StoreKey,
		epochstypes.StoreKey,
		inflationtypes.StoreKey,
		sudotypes.StoreKey,
		wasmtypes.StoreKey,
		devgastypes.StoreKey,
		tokenfactorytypes.StoreKey,

		evm.StoreKey,
	)
	app.tkeys = sdk.NewTransientStoreKeys(paramstypes.TStoreKey, evm.TransientKey)
	app.memKeys = sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	for _, k := range app.keys {
		if err := app.RegisterStores(k); err != nil {
			panic(err)
		}
	}
	for _, k := range app.tkeys {
		if err := app.RegisterStores(k); err != nil {
			panic(err)
		}
	}
	for _, k := range app.memKeys {
		if err := app.RegisterStores(k); err != nil {
			panic(err)
		}
	}

	wasmConfig := app.initNonDepinjectKeepers(appOpts)

	// register non-depinject modules
	if err := app.RegisterModules(
		// core modules
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx, app.txConfig),
		capability.NewAppModule(app.appCodec, *app.capabilityKeeper, false),
		feegrantmodule.NewAppModule(app.appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(app.appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.getSubspace(govtypes.ModuleName)),
		slashing.NewAppModule(app.appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.getSubspace(slashingtypes.ModuleName)),
		upgrade.NewAppModule(&app.upgradeKeeper),
		params.NewAppModule(app.paramsKeeper),
		authzmodule.NewAppModule(app.appCodec, app.authzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),

		// Nibiru modules
		oracle.NewAppModule(app.appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper, app.SudoKeeper),
		epochs.NewAppModule(app.appCodec, app.EpochsKeeper),
		inflation.NewAppModule(app.InflationKeeper, app.AccountKeeper, *app.StakingKeeper),
		sudo.NewAppModule(app.appCodec, app.SudoKeeper),
		genmsg.NewAppModule(app.MsgServiceRouter()),

		// ibc
		evidence.NewAppModule(app.evidenceKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		ibctransfer.NewAppModule(app.ibcTransferKeeper),
		ibcfee.NewAppModule(app.ibcFeeKeeper),
		ica.NewAppModule(&app.icaControllerKeeper, &app.icaHostKeeper),
		ibcwasm.NewAppModule(app.WasmClientKeeper),

		// evm
		evmmodule.NewAppModule(app.EvmKeeper, app.AccountKeeper),

		// wasm
		wasm.NewAppModule(app.appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.getSubspace(wasmtypes.ModuleName)),
		devgas.NewAppModule(app.DevGasKeeper, app.AccountKeeper),
		tokenfactory.NewAppModule(app.TokenFactoryKeeper, app.AccountKeeper),

		crisis.NewAppModule(&app.crisisKeeper, cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants)), app.getSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
	); err != nil {
		panic(err)
	}

	// make sure to get the ibc tendermint client interface types
	ibctm.AppModuleBasic{}.RegisterInterfaces(app.interfaceRegistry)
	ibctm.AppModuleBasic{}.RegisterLegacyAminoCodec(app.legacyAmino)

	// make sure to register the eth crypto codec types
	cryptocodec.RegisterInterfaces(app.interfaceRegistry)
	cryptocodec.RegisterCrypto(app.legacyAmino)

	// load state streaming if enabled
	if _, _, err := streaming.LoadStreamingServices(app.App.BaseApp, appOpts, app.appCodec, logger, app.kvStoreKeys()); err != nil {
		logger.Error("failed to load state streaming", "err", err)
		os.Exit(1)
	}

	/****  Module Options ****/

	app.ModuleManager.RegisterInvariants(&app.crisisKeeper)

	app.setupUpgrades()

	// add test gRPC service for testing gRPC queries in isolation
	testdata.RegisterQueryServer(app.GRPCQueryRouter(), testdata.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.getSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()

	// A custom InitChainer can be set if extra pre-init-genesis logic is required.
	// By default, when using app wiring enabled module, this is not required.
	// For instance, the upgrade module will set automatically the module version map in its init genesis thanks to app wiring.
	// However, when registering a module manually (i.e. that does not support app wiring), the module version map
	// must be set manually as follow. The upgrade module will de-duplicate the module version map.
	//
	// app.SetInitChainer(func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	// 	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	// 	return app.App.InitChainer(ctx, req)
	// })

	// initialize custom antehandler
	app.SetAnteHandler(NewAnteHandler(app.AppKeepers, ante.AnteHandlerOptions{
		HandlerOptions: authante.HandlerOptions{
			AccountKeeper:          app.AccountKeeper,
			BankKeeper:             app.BankKeeper,
			FeegrantKeeper:         app.FeeGrantKeeper,
			SignModeHandler:        app.txConfig.SignModeHandler(),
			SigGasConsumer:         authante.DefaultSigVerificationGasConsumer,
			ExtensionOptionChecker: func(*codectypes.Any) bool { return true },
		},
		IBCKeeper:         app.ibcKeeper,
		TxCounterStoreKey: app.keys[wasmtypes.StoreKey],
		WasmConfig:        &wasmConfig,
		DevGasKeeper:      &app.DevGasKeeper,
		DevGasBankKeeper:  app.BankKeeper,
		// TODO: feat(evm): enable app/server/config flag for Evm MaxTxGasWanted.
		MaxTxGasWanted: DefaultMaxTxGasWanted,
		EvmKeeper:      app.EvmKeeper,
		AccountKeeper:  app.AccountKeeper,
	}))

	// register snapshot extensions
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

	if loadLatest {
		// Initialize pinned codes in wasmvm as they are not persisted there
		if err := ibcwasmkeeper.InitializePinnedCodes(app.BaseApp.NewUncachedContext(true, cmtproto.Header{}), app.appCodec); err != nil {
			cmtos.Exit(fmt.Sprintf("failed to initialize pinned codes %s", err))
		}
	}

	return app
}

// Name returns the name of the App
func (app *NibiruApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *NibiruApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.ModuleManager.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *NibiruApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.ModuleManager.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *NibiruApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.upgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
}

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

func (app *NibiruApp) kvStoreKeys() map[string]*storetypes.KVStoreKey {
	keys := make(map[string]*storetypes.KVStoreKey)
	for _, k := range app.GetStoreKeys() {
		if kv, ok := k.(*storetypes.KVStoreKey); ok {
			keys[kv.Name()] = kv
		}
	}

	return keys
}

// getSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruApp) getSubspace(moduleName string) paramstypes.Subspace {
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
	return app.txConfig
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
