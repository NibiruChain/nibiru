package app

import (
	"path/filepath"

	ibcmock "github.com/cosmos/ibc-go/v4/testing/mock"

	"github.com/cosmos/cosmos-sdk/baseapp"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"

	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"

	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"

	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	ibcfee "github.com/cosmos/ibc-go/v4/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v4/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v4/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v4/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v4/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v4/modules/core/keeper"

	"github.com/NibiruChain/nibiru/x/epochs"
	epochskeeper "github.com/NibiruChain/nibiru/x/epochs/keeper"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/sudo"

	"github.com/NibiruChain/nibiru/x/inflation"
	inflationkeeper "github.com/NibiruChain/nibiru/x/inflation/keeper"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"

	oracle "github.com/NibiruChain/nibiru/x/oracle"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"

	perpamm "github.com/NibiruChain/nibiru/x/perp/amm"
	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	v2perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	perp "github.com/NibiruChain/nibiru/x/perp/module/v1"

	perptypes "github.com/NibiruChain/nibiru/x/perp/types/v1"
	v2perptypes "github.com/NibiruChain/nibiru/x/perp/types/v2"
	"github.com/NibiruChain/nibiru/x/spot"
	spotkeeper "github.com/NibiruChain/nibiru/x/spot/keeper"
	spottypes "github.com/NibiruChain/nibiru/x/spot/types"

	"github.com/NibiruChain/nibiru/x/stablecoin"
	stablecoinkeeper "github.com/NibiruChain/nibiru/x/stablecoin/keeper"
	stablecointypes "github.com/NibiruChain/nibiru/x/stablecoin/types"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func GetStoreKeys() (
	keys map[string]*sdk.KVStoreKey,
	tkeys map[string]*sdk.TransientStoreKey,
	memKeys map[string]*sdk.MemoryStoreKey,
) {
	keys = sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		upgradetypes.StoreKey,
		feegrant.StoreKey,
		evidencetypes.StoreKey,
		capabilitytypes.StoreKey,
		authzkeeper.StoreKey,

		// ibc keys
		ibchost.StoreKey,
		ibctransfertypes.StoreKey,
		ibcfeetypes.StoreKey,

		// nibiru x/ keys
		spottypes.StoreKey,
		stablecointypes.StoreKey,
		oracletypes.StoreKey,
		epochstypes.StoreKey,
		perptypes.StoreKey,
		v2perptypes.StoreKey,
		perpammtypes.StoreKey,
		inflationtypes.StoreKey,
		sudo.StoreKey,
		wasm.StoreKey,
	)
	tkeys = sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys = sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, stablecointypes.MemStoreKey)
	return keys, tkeys, memKeys
}

func (app *NibiruApp) InitKeepers(
	skipUpgradeHeights map[int64]bool,
	homePath string,
	appOpts servertypes.AppOptions,
) (wasmConfig wasmtypes.WasmConfig) {
	var appCodec codec.Codec = app.appCodec
	var legacyAmino *codec.LegacyAmino = app.legacyAmino
	var bApp *baseapp.BaseApp = app.BaseApp

	keys := app.keys
	tkeys := app.tkeys
	memKeys := app.memKeys

	app.paramsKeeper = initParamsKeeper(
		appCodec, legacyAmino, keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	bApp.SetParamStore(app.paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramskeeper.ConsensusParamsKeyTable()))

	/* Add capabilityKeeper and ScopeToModule for the ibc module
	   This allows authentication of object-capability permissions for each of
	   the IBC channels.
	*/
	app.capabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	app.ScopedIBCKeeper = app.capabilityKeeper.ScopeToModule(ibchost.ModuleName)
	scopedFeeMockKeeper := app.capabilityKeeper.ScopeToModule(MockFeePort)
	app.ScopedTransferKeeper = app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	_ = app.capabilityKeeper.ScopeToModule(ibcmock.ModuleName)

	// seal capability keeper after scoping modules
	//app.capabilityKeeper.Seal()

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], app.GetSubspace(authtypes.ModuleName), authtypes.ProtoBaseAccount, maccPerms,
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.AccountKeeper, app.GetSubspace(banktypes.ModuleName), app.ModuleAccountAddrs(),
	)
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey], app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName),
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec, keys[distrtypes.StoreKey], app.GetSubspace(distrtypes.ModuleName), app.AccountKeeper, app.BankKeeper,
		&stakingKeeper, authtypes.FeeCollectorName, app.ModuleAccountAddrs(),
	)
	app.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec, keys[slashingtypes.StoreKey], &stakingKeeper, app.GetSubspace(slashingtypes.ModuleName),
	)

	invCheckPeriod := app.invCheckPeriod
	app.crisisKeeper = crisiskeeper.NewKeeper(
		app.GetSubspace(crisistypes.ModuleName), invCheckPeriod, app.BankKeeper, authtypes.FeeCollectorName,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, keys[feegrant.StoreKey], app.AccountKeeper)

	/*upgradeKeeper must be created before ibcKeeper. */
	app.upgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		app.BaseApp)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.stakingKeeper = *stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.slashingKeeper.Hooks()),
	)

	app.authzKeeper = authzkeeper.NewKeeper(keys[authzkeeper.StoreKey], appCodec, app.BaseApp.MsgServiceRouter())

	// ---------------------------------- Nibiru Chain x/ keepers

	app.SpotKeeper = spotkeeper.NewKeeper(
		appCodec, keys[spottypes.StoreKey], app.GetSubspace(spottypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper)

	app.OracleKeeper = oraclekeeper.NewKeeper(appCodec, keys[oracletypes.StoreKey], app.GetSubspace(oracletypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.stakingKeeper, distrtypes.ModuleName,
	)

	app.StablecoinKeeper = stablecoinkeeper.NewKeeper(
		appCodec, keys[stablecointypes.StoreKey], memKeys[stablecointypes.MemStoreKey],
		app.GetSubspace(stablecointypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.OracleKeeper, app.SpotKeeper,
	)

	app.PerpAmmKeeper = perpammkeeper.NewKeeper(
		appCodec,
		keys[perpammtypes.StoreKey],
		app.OracleKeeper,
	)

	app.EpochsKeeper = epochskeeper.NewKeeper(
		appCodec, keys[epochstypes.StoreKey],
	)

	app.PerpKeeper = perpkeeper.NewKeeper(
		appCodec, keys[perptypes.StoreKey],
		app.GetSubspace(perptypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.OracleKeeper, app.PerpAmmKeeper, app.EpochsKeeper,
	)

	app.PerpKeeperV2 = v2perpkeeper.NewKeeper(appCodec, keys[v2perptypes.StoreKey], app.GetSubspace(v2perptypes.ModuleName), app.AccountKeeper, app.BankKeeper, app.OracleKeeper, app.EpochsKeeper)

	app.InflationKeeper = inflationkeeper.NewKeeper(
		appCodec, keys[inflationtypes.StoreKey], app.GetSubspace(inflationtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.stakingKeeper, authtypes.FeeCollectorName,
	)

	app.SudoKeeper = sudo.NewKeeper(
		appCodec, keys[sudo.StoreKey],
	)

	app.EpochsKeeper.SetHooks(
		epochstypes.NewMultiEpochHooks(app.StablecoinKeeper.Hooks(), app.PerpKeeper.Hooks(), app.InflationKeeper.Hooks()),
	)

	// ---------------------------------- IBC keepers

	app.ibcKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibchost.StoreKey],
		app.GetSubspace(ibchost.ModuleName),
		app.stakingKeeper,
		app.upgradeKeeper,
		app.ScopedIBCKeeper,
	)

	// IBC Fee Module keeper
	app.ibcFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, keys[ibcfeetypes.StoreKey], app.GetSubspace(ibcfeetypes.ModuleName),
		app.ibcKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper, app.AccountKeeper, app.BankKeeper,
	)

	app.ScopedWasmKeeper = app.capabilityKeeper.ScopeToModule(wasm.ModuleName)

	wasmDir := filepath.Join(homePath, "data")
	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	//
	// NOTE: This keeper depends on all of pointers to the the Keepers to which
	// it binds. Thus, it must be instantiated after those keepers have been
	// assigned.
	// For example, if there are bindings for the x/perp module, then the app
	// passed to GetWasmOpts must already have a non-nil PerpKeeper.
	supportedFeatures := "iterator,staking,stargate"
	app.WasmKeeper = wasm.NewKeeper(
		appCodec,
		keys[wasm.StoreKey],
		app.GetSubspace(wasm.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		app.DistrKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.ScopedWasmKeeper,
		app.transferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		supportedFeatures,
		GetWasmOpts(*app, appOpts)...,
	)

	// register the proposal types

	govRouter := govtypes.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.DistrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(app.upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.ibcKeeper.ClientKeeper)).
		AddRoute(perpammtypes.RouterKey, perpamm.NewMarketProposalHandler(app.PerpAmmKeeper))

	// Create evidence keeper.
	// This keeper automatically includes an evidence router.
	app.evidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], &app.stakingKeeper,
		app.slashingKeeper,
	)

	/* Create IBC module and a static IBC router */
	ibcRouter := porttypes.NewRouter()

	app.transferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		/* paramSubspace */ app.GetSubspace(ibctransfertypes.ModuleName),
		/* ibctransfertypes.ICS4Wrapper */ app.ibcFeeKeeper,
		/* ibctransfertypes.ChannelKeeper */ app.ibcKeeper.ChannelKeeper,
		/* ibctransfertypes.PortKeeper */ &app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
	)

	// Mock Module setup for testing IBC and also acts as the interchain accounts authentication module
	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	mockModule := ibcmock.NewAppModule(&app.ibcKeeper.PortKeeper)

	// Create Transfer Stack
	// SendPacket, since it is originating from the application to core IBC:
	// transferKeeper.SendPacket -> fee.SendPacket -> channel.SendPacket

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
	// channel.RecvPacket -> fee.OnRecvPacket -> transfer.OnRecvPacket

	// transfer stack contains (from top to bottom):
	// - IBC Fee Middleware
	// - Transfer

	// create IBC module from bottom to top of stack
	var transferStack porttypes.IBCModule
	transferStack = ibctransfer.NewIBCModule(app.transferKeeper)
	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.ibcFeeKeeper)

	// Add transfer stack to IBC Router
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)

	// Create Mock IBC Fee module stack for testing
	// SendPacket, since it is originating from the application to core IBC:
	// mockModule.SendPacket -> fee.SendPacket -> channel.SendPacket

	// OnRecvPacket, message that originates from core IBC and goes down to app, the flow is the otherway
	// channel.RecvPacket -> fee.OnRecvPacket -> mockModule.OnRecvPacket

	// OnAcknowledgementPacket as this is where fee's are paid out
	// mockModule.OnAcknowledgementPacket -> fee.OnAcknowledgementPacket -> channel.OnAcknowledgementPacket

	// create fee wrapped mock module
	feeMockModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewMockIBCApp(MockFeePort, scopedFeeMockKeeper))
	app.FeeMockModule = feeMockModule
	feeWithMockModule := ibcfee.NewIBCMiddleware(feeMockModule, app.ibcFeeKeeper)
	ibcRouter.AddRoute(MockFeePort, feeWithMockModule)

	/* SetRouter finalizes all routes by sealing the router.
	   No more routes can be added. */
	app.ibcKeeper.SetRouter(ibcRouter)

	app.GovKeeper = govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.GetSubspace(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, &app.stakingKeeper, govRouter,
	)

	return wasmConfig
}

func (app *NibiruApp) AppModules(
	encodingConfig simappparams.EncodingConfig,
	skipGenesisInvariants bool,
) []module.AppModule {
	appCodec := app.appCodec
	spotModule := spot.NewAppModule(
		appCodec, app.SpotKeeper, app.AccountKeeper, app.BankKeeper)
	oracleModule := oracle.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper)
	epochsModule := epochs.NewAppModule(appCodec, app.EpochsKeeper)
	stablecoinModule := stablecoin.NewAppModule(
		appCodec, app.StablecoinKeeper, app.AccountKeeper, app.BankKeeper,
		nil,
	)
	perpModule := perp.NewAppModule(
		appCodec, app.PerpKeeper, app.AccountKeeper, app.BankKeeper,
		app.OracleKeeper,
	)
	perpAmmModule := perpamm.NewAppModule(
		appCodec, app.PerpAmmKeeper, app.OracleKeeper,
	)
	inflationModule := inflation.NewAppModule(
		app.InflationKeeper, app.AccountKeeper, app.stakingKeeper,
	)
	sudoModule := sudo.NewAppModule(appCodec, app.SudoKeeper)

	return []module.AppModule{
		genutil.NewAppModule(
			app.AccountKeeper, app.stakingKeeper, app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.capabilityKeeper),
		crisis.NewAppModule(&app.crisisKeeper, skipGenesisInvariants),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		slashing.NewAppModule(appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
		staking.NewAppModule(appCodec, app.stakingKeeper, app.AccountKeeper, app.BankKeeper),
		upgrade.NewAppModule(app.upgradeKeeper),
		params.NewAppModule(app.paramsKeeper),
		authzmodule.NewAppModule(appCodec, app.authzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),

		// native x/
		spotModule,
		stablecoinModule,
		oracleModule,
		epochsModule,
		perpAmmModule,
		perpModule,
		inflationModule,
		sudoModule,

		// ibc
		evidence.NewAppModule(app.evidenceKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		ibctransfer.NewAppModule(app.transferKeeper),
		ibcfee.NewAppModule(app.ibcFeeKeeper),

		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.stakingKeeper, app.AccountKeeper, app.BankKeeper),
	}
}

// OrderedModuleNames: Module names ordered for the begin and end block hooks
func OrderedModuleNames() []string {
	return []string{
		// --------------------------------------------------------------------
		// Cosmos-SDK modules
		//
		// NOTE (InitGenesis requirement): Capability module must occur
		//   first so that it can initialize any capabilities, allowing other
		//   modules that want to create or claim capabilities afterwards in
		//   "InitChain" safely.
		// NOTE (BeginBlocker requirement): Capability module's beginblocker
		//   must come before any modules using capabilities (e.g. IBC)
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		// NOTE (BeginBlocker requirement): During begin block, x/slashing must
		//   come after x/distribution so that there won't be anything left over
		//   in the validator pool. This makes sure that "CanWithdrawInvariant"
		//   remains invariant.
		distrtypes.ModuleName,
		// NOTE (BeginBlocker requirement): staking module is required if
		//   HistoricalEntries param > 0
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		crisistypes.ModuleName,
		govtypes.ModuleName,
		genutiltypes.ModuleName,
		// NOTE (SetOrderInitGenesis requirement): genutils must occur after
		//   staking so that pools will be properly initialized with tokens from
		//   genesis accounts.
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,

		// --------------------------------------------------------------------
		// Native x/ Modules
		epochstypes.ModuleName,
		stablecointypes.ModuleName,
		spottypes.ModuleName,
		oracletypes.ModuleName,
		perpammtypes.ModuleName,
		perptypes.ModuleName,
		v2perptypes.ModuleName,
		inflationtypes.ModuleName,
		sudo.ModuleName,

		// --------------------------------------------------------------------
		// IBC modules
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,

		// --------------------------------------------------------------------
		// CosmWasm
		wasm.ModuleName,
	}
}

// NOTE: Any module instantiated in the module manager that is later modified
// must be passed by reference here.
func (app *NibiruApp) InitModuleManager(
	encodingConfig simappparams.EncodingConfig,
	skipGenesisInvariants bool,
) {
	app.mm = module.NewManager(
		app.AppModules(encodingConfig, skipGenesisInvariants)...,
	)

	// Init module orders for hooks and genesis
	moduleNames := OrderedModuleNames()
	app.mm.SetOrderBeginBlockers(moduleNames...)
	app.mm.SetOrderEndBlockers(moduleNames...)
	app.mm.SetOrderInitGenesis(moduleNames...)

	// Uncomment if you want to set a custom migration order here.
	// app.mm.SetOrderMigrations(custom order)

	app.mm.RegisterInvariants(&app.crisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)
	app.configurator = module.NewConfigurator(
		app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)
}

func (app *NibiruApp) InitSimulationManager(
	appCodec codec.Codec,
) {
	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	epochsModule := epochs.NewAppModule(appCodec, app.EpochsKeeper)
	app.sm = module.NewSimulationManager(
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		staking.NewAppModule(appCodec, app.stakingKeeper, app.AccountKeeper, app.BankKeeper),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
		slashing.NewAppModule(appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
		params.NewAppModule(app.paramsKeeper),
		authzmodule.NewAppModule(appCodec, app.authzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		// native x/
		epochsModule,
		// ibc
		capability.NewAppModule(appCodec, *app.capabilityKeeper),
		evidence.NewAppModule(app.evidenceKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		ibctransfer.NewAppModule(app.transferKeeper),
		ibcfee.NewAppModule(app.ibcFeeKeeper),
	)

	app.sm.RegisterStoreDecoders()
}
