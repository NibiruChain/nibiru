package app

import (
	"github.com/NibiruChain/nibiru/x/sudo/keeper"
	"path/filepath"

	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"

	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
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
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
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
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/spf13/cast"

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

	perpv2keeper "github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	perpv2 "github.com/NibiruChain/nibiru/x/perp/v2/module"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"

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
	keys map[string]*types.KVStoreKey,
	tkeys map[string]*types.TransientStoreKey,
	memKeys map[string]*types.MemoryStoreKey,
) {
	keys = sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
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

		// nibiru x/ keys
		spottypes.StoreKey,
		stablecointypes.StoreKey,
		oracletypes.StoreKey,
		epochstypes.StoreKey,
		perpv2types.StoreKey,
		inflationtypes.StoreKey,
		sudotypes.StoreKey,
		wasm.StoreKey,
	)
	tkeys = sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys = sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, stablecointypes.MemStoreKey)
	return keys, tkeys, memKeys
}

func (app *NibiruApp) InitKeepers(
	appOpts servertypes.AppOptions,
) (wasmConfig wasmtypes.WasmConfig) {
	var appCodec = app.appCodec
	var legacyAmino = app.legacyAmino
	var bApp = app.BaseApp

	keys := app.keys
	tkeys := app.tkeys
	memKeys := app.memKeys

	app.paramsKeeper = initParamsKeeper(
		appCodec, legacyAmino, keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, keys[consensusparamtypes.StoreKey], authtypes.NewModuleAddress(govtypes.ModuleName).String())
	bApp.SetParamStore(&app.ConsensusParamsKeeper)

	/* Add capabilityKeeper and ScopeToModule for the ibc module
	   This allows authentication of object-capability permissions for each of
	   the IBC channels.
	*/
	app.capabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)
	app.ScopedIBCKeeper = app.capabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	//scopedFeeMockKeeper := app.capabilityKeeper.ScopeToModule(MockFeePort)
	app.ScopedTransferKeeper = app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	_ = app.capabilityKeeper.ScopeToModule(ibcmock.ModuleName)

	// seal capability keeper after scoping modules
	//app.capabilityKeeper.Seal()

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey],
		authtypes.ProtoBaseAccount,
		maccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		app.AccountKeeper,
		BlockedAddresses(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.stakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		keys[distrtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	app.crisisKeeper = *crisiskeeper.NewKeeper(
		appCodec,
		keys[crisistypes.StoreKey],
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, keys[feegrant.StoreKey], app.AccountKeeper)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))

	/*upgradeKeeper must be created before ibcKeeper. */
	app.upgradeKeeper = *upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		keys[slashingtypes.StoreKey],
		app.stakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.slashingKeeper.Hooks()),
	)

	app.authzKeeper = authzkeeper.NewKeeper(
		keys[authzkeeper.StoreKey],
		appCodec,
		app.BaseApp.MsgServiceRouter(),
		app.AccountKeeper,
	)

	// ---------------------------------- Nibiru Chain x/ keepers

	app.SpotKeeper = spotkeeper.NewKeeper(
		appCodec, keys[spottypes.StoreKey], app.GetSubspace(spottypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper)

	app.OracleKeeper = oraclekeeper.NewKeeper(appCodec, keys[oracletypes.StoreKey],
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.stakingKeeper, distrtypes.ModuleName,
	)

	app.StablecoinKeeper = stablecoinkeeper.NewKeeper(
		appCodec, keys[stablecointypes.StoreKey], memKeys[stablecointypes.MemStoreKey],
		app.GetSubspace(stablecointypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.OracleKeeper, app.SpotKeeper,
	)

	app.EpochsKeeper = epochskeeper.NewKeeper(
		appCodec, keys[epochstypes.StoreKey],
	)

	app.PerpKeeperV2 = perpv2keeper.NewKeeper(
		appCodec, keys[perpv2types.StoreKey],
		app.AccountKeeper, app.BankKeeper, app.OracleKeeper, app.EpochsKeeper,
	)

	app.InflationKeeper = inflationkeeper.NewKeeper(
		appCodec, keys[inflationtypes.StoreKey], app.GetSubspace(inflationtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.stakingKeeper, authtypes.FeeCollectorName,
	)

	app.SudoKeeper = keeper.NewKeeper(
		appCodec, keys[sudotypes.StoreKey],
	)

	app.EpochsKeeper.SetHooks(
		epochstypes.NewMultiEpochHooks(
			app.StablecoinKeeper.Hooks(),
			app.PerpKeeperV2.Hooks(),
			app.InflationKeeper.Hooks(),
			app.OracleKeeper.Hooks(),
		),
	)

	// ---------------------------------- IBC keepers

	app.ibcKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.stakingKeeper,
		app.upgradeKeeper,
		app.ScopedIBCKeeper,
	)

	// IBC Fee Module keeper
	app.ibcFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, keys[ibcfeetypes.StoreKey],
		app.ibcKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
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
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.ibcFeeKeeper, // ISC4 Wrapper: fee IBC middleware
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.ScopedWasmKeeper,
		app.transferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		supportedFeatures,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		GetWasmOpts(*app, appOpts)...,
	)

	// register the proposal types

	// Create evidence keeper.
	// This keeper automatically includes an evidence router.
	app.evidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], app.stakingKeeper,
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
	//mockModule := ibcmock.NewAppModule(&app.ibcKeeper.PortKeeper)

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
	//feeMockModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewMockIBCApp(MockFeePort, scopedFeeMockKeeper))
	//app.FeeMockModule = feeMockModule
	//feeWithMockModule := ibcfee.NewIBCMiddleware(feeMockModule, app.ibcFeeKeeper)
	//ibcRouter.AddRoute(MockFeePort, feeWithMockModule)

	/* SetRouter finalizes all routes by sealing the router.
	   No more routes can be added. */
	app.ibcKeeper.SetRouter(ibcRouter)

	govRouter := govv1beta1types.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govv1beta1types.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		//AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.DistrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&app.upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.ibcKeeper.ClientKeeper))

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		keys[govtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		app.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	govKeeper.SetLegacyRouter(govRouter)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(),
	)

	return wasmConfig
}

func (app *NibiruApp) AppModules(
	encodingConfig EncodingConfig,
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
	perpv2Module := perpv2.NewAppModule(
		appCodec, app.PerpKeeperV2, app.AccountKeeper, app.BankKeeper, app.OracleKeeper,
	)
	inflationModule := inflation.NewAppModule(
		app.InflationKeeper, app.AccountKeeper, *app.stakingKeeper,
	)
	sudoModule := sudo.NewAppModule(appCodec, app.SudoKeeper)

	return []module.AppModule{
		genutil.NewAppModule(
			app.AccountKeeper, app.stakingKeeper, app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.capabilityKeeper, false),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		// TODO dont we have mint module?
		slashing.NewAppModule(appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.stakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(&app.upgradeKeeper),
		params.NewAppModule(app.paramsKeeper),
		authzmodule.NewAppModule(appCodec, app.authzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),

		// native x/
		spotModule,
		stablecoinModule,
		oracleModule,
		epochsModule,
		perpv2Module,
		inflationModule,
		sudoModule,
		perpv2Module,

		// ibc
		evidence.NewAppModule(app.evidenceKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		ibctransfer.NewAppModule(app.transferKeeper),
		ibcfee.NewAppModule(app.ibcFeeKeeper),

		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.stakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		crisis.NewAppModule(&app.crisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
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
		perpv2types.ModuleName,
		inflationtypes.ModuleName,
		sudotypes.ModuleName,

		// --------------------------------------------------------------------
		// IBC modules
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		ibcfeetypes.ModuleName,

		// --------------------------------------------------------------------
		// CosmWasm
		wasm.ModuleName,
	}
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range GetMaccPerms() {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	// allow the following addresses to receive funds
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	return modAccAddrs
}

// InitModuleManager Load all the modules and stores them in the module manager
// NOTE: Any module instantiated in the module manager that is later modified
// must be passed by reference here.
func (app *NibiruApp) InitModuleManager(
	encodingConfig EncodingConfig,
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
	app.configurator = module.NewConfigurator(
		app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	// see https://github.com/cosmos/cosmos-sdk/blob/666c345ad23ddda9523cc5cd1b71187d91c26f34/simapp/upgrades.go#L35-L57
	for _, subspace := range app.paramsKeeper.GetSubspaces() {
		switch subspace.Name() {
		case authtypes.ModuleName:
			subspace.WithKeyTable(authtypes.ParamKeyTable()) //nolint:staticcheck
		case banktypes.ModuleName:
			subspace.WithKeyTable(banktypes.ParamKeyTable()) //nolint:staticcheck
		case stakingtypes.ModuleName:
			subspace.WithKeyTable(stakingtypes.ParamKeyTable()) //nolint:staticcheck
		case distrtypes.ModuleName:
			subspace.WithKeyTable(distrtypes.ParamKeyTable()) //nolint:staticcheck
		case slashingtypes.ModuleName:
			subspace.WithKeyTable(slashingtypes.ParamKeyTable()) //nolint:staticcheck
		case govtypes.ModuleName:
			subspace.WithKeyTable(govv1types.ParamKeyTable()) //nolint:staticcheck
		case crisistypes.ModuleName:
			subspace.WithKeyTable(crisistypes.ParamKeyTable()) //nolint:staticcheck
		}
	}
}

// TODO: Simulation manager
func (app *NibiruApp) InitSimulationManager(
	appCodec codec.Codec,
) {
	//// create the simulation manager and define the order of the modules for deterministic simulations
	////
	//// NOTE: this is not required apps that don't use the simulator for fuzz testing
	//// transactions
	//epochsModule := epochs.NewAppModule(appCodec, app.EpochsKeeper)
	//app.sm = module.NewSimulationManager(
	//	auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
	//	bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
	//	feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
	//	gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
	//	staking.NewAppModule(appCodec, app.stakingKeeper, app.AccountKeeper, app.BankKeeper),
	//	distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
	//	slashing.NewAppModule(appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
	//	params.NewAppModule(app.paramsKeeper),
	//	authzmodule.NewAppModule(appCodec, app.authzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
	//	// native x/
	//	epochsModule,
	//	// ibc
	//	capability.NewAppModule(appCodec, *app.capabilityKeeper),
	//	evidence.NewAppModule(app.evidenceKeeper),
	//	ibc.NewAppModule(app.ibcKeeper),
	//	ibctransfer.NewAppModule(app.transferKeeper),
	//	ibcfee.NewAppModule(app.ibcFeeKeeper),
	//)
	//
	//app.sm.RegisterStoreDecoders()
}
