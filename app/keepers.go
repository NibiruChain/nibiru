package app

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclientclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/client"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
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
)

const wasmVmContractMemoryLimit = 32

type AppKeepers struct {
	keepers.PublicKeepers
	privateKeepers
}

type privateKeepers struct {
	capabilityKeeper *capabilitykeeper.Keeper
	slashingKeeper   slashingkeeper.Keeper
	crisisKeeper     *crisiskeeper.Keeper
	upgradeKeeper    *upgradekeeper.Keeper
	paramsKeeper     paramskeeper.Keeper
	authzKeeper      authzkeeper.Keeper

	// --------------------------------------------------------------------
	// IBC keepers
	// --------------------------------------------------------------------
	/* evidenceKeeper is responsible for managing persistence, state transitions
	   and query handling for the evidence module. It is required to set up
	   the IBC light client misbehavior evidence route. */
	evidenceKeeper evidencekeeper.Keeper

	/* ibcKeeper defines each ICS keeper for IBC. ibcKeeper must be a pointer in
	   the app, so we can SetRouter on it correctly. */
	ibcKeeper    *ibckeeper.Keeper
	ibcFeeKeeper ibcfeekeeper.Keeper
	/* ibcTransferKeeper is for cross-chain fungible token transfers. */
	ibcTransferKeeper   ibctransferkeeper.Keeper
	icaControllerKeeper icacontrollerkeeper.Keeper
	icaHostKeeper       icahostkeeper.Keeper
}

func initStoreKeys() (
	keys map[string]*types.KVStoreKey,
	tkeys map[string]*types.TransientStoreKey,
	memKeys map[string]*types.MemoryStoreKey,
) {
	keys = sdk.NewKVStoreKeys(

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
	tkeys = sdk.NewTransientStoreKeys(paramstypes.TStoreKey, evm.TransientKey)
	memKeys = sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	return keys, tkeys, memKeys
}

// orderedModuleNames: Module names ordered for the begin and end block hooks
func orderedModuleNames() []string {
	return []string{
		// --------------------------------------------------------------------
		// Cosmos-SDK modules
		//
		// NOTE: (BeginBlocker requirement): upgrade module must occur first
		upgradetypes.ModuleName,

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

		// --------------------------------------------------------------------
		// Native x/ Modules
		epochstypes.ModuleName,
		oracletypes.ModuleName,
		inflationtypes.ModuleName,
		sudotypes.ModuleName,

		// --------------------------------------------------------------------
		// IBC modules
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		ibcfeetypes.ModuleName,
		icatypes.ModuleName,
		ibcwasmtypes.ModuleName,

		// --------------------------------------------------------------------
		evm.ModuleName,

		// --------------------------------------------------------------------
		// CosmWasm
		wasmtypes.ModuleName,
		devgastypes.ModuleName,
		tokenfactorytypes.ModuleName,

		// Everything else should be before genmsg
		genmsg.ModuleName,
	}
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	// allow the following addresses to receive funds
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	return modAccAddrs
}

// // initModuleManager Load all the modules and stores them in the module manager
// // NOTE: Any module instantiated in the module manager that is later modified
// // must be passed by reference here.
// func (app *NibiruApp) initModuleManager(
// 	encodingConfig EncodingConfig,
// 	skipGenesisInvariants bool,
// ) {
// 	app.ModuleManager = module.NewManager(
// 		app.initAppModules(encodingConfig, skipGenesisInvariants)...,
// 	)

// 	orderedModules := orderedModuleNames()
// 	app.ModuleManager.SetOrderBeginBlockers(orderedModules...)
// 	app.ModuleManager.SetOrderEndBlockers(orderedModules...)
// 	app.ModuleManager.SetOrderInitGenesis(orderedModules...)
// 	app.ModuleManager.SetOrderExportGenesis(orderedModules...)

// 	// Uncomment if you want to set a custom migration order here.
// 	// app.mm.SetOrderMigrations(custom order)

// 	app.ModuleManager.RegisterInvariants(app.crisisKeeper)
// 	app.configurator = module.NewConfigurator(
// 		app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
// 	app.ModuleManager.RegisterServices(app.configurator)

// 	// see https://github.com/cosmos/cosmos-sdk/blob/666c345ad23ddda9523cc5cd1b71187d91c26f34/simapp/upgrades.go#L35-L57
// 	for _, subspace := range app.paramsKeeper.GetSubspaces() {
// 		switch subspace.Name() {
// 		case authtypes.ModuleName:
// 			subspace.WithKeyTable(authtypes.ParamKeyTable()) //nolint:staticcheck
// 		case banktypes.ModuleName:
// 			subspace.WithKeyTable(banktypes.ParamKeyTable()) //nolint:staticcheck
// 		case stakingtypes.ModuleName:
// 			subspace.WithKeyTable(stakingtypes.ParamKeyTable()) //nolint:staticcheck
// 		case distrtypes.ModuleName:
// 			subspace.WithKeyTable(distrtypes.ParamKeyTable()) //nolint:staticcheck
// 		case slashingtypes.ModuleName:
// 			subspace.WithKeyTable(slashingtypes.ParamKeyTable()) //nolint:staticcheck
// 		case govtypes.ModuleName:
// 			subspace.WithKeyTable(govv1types.ParamKeyTable()) //nolint:staticcheck
// 		case crisistypes.ModuleName:
// 			subspace.WithKeyTable(crisistypes.ParamKeyTable()) //nolint:staticcheck
// 		}
// 	}
// }

// ModuleBasicManager The app's collection of module.AppModuleBasic
// implementations. These set up non-dependant module elements, such as codec
// registration and genesis verification.
func ModuleBasicManager() module.BasicManager {
	return module.NewBasicManager(
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
		// native x/
		evmmodule.AppModuleBasic{},
		oracle.AppModuleBasic{},
		epochs.AppModuleBasic{},
		inflation.AppModuleBasic{},
		sudo.AppModuleBasic{},
		wasm.AppModuleBasic{},
		devgas.AppModuleBasic{},
		tokenfactory.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
		genmsg.AppModule{},
	)
}

func ModuleAccPerms() map[string][]string {
	return map[string][]string{
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
}

func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key,
	tkey storetypes.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)
	// Nibiru core params keepers | x/
	paramsKeeper.Subspace(epochstypes.ModuleName)
	paramsKeeper.Subspace(inflationtypes.ModuleName)
	// ibc params keepers
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(ibcfeetypes.ModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	// wasm params keepers
	paramsKeeper.Subspace(wasmtypes.ModuleName)
	paramsKeeper.Subspace(devgastypes.ModuleName)

	return paramsKeeper
}

func (app *NibiruApp) initSimulationManager(
	appCodec codec.Codec,
) {
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()
}
