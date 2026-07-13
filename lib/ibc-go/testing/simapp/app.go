package simapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/baseapp"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/docs/statik" // this is used for serving docs
	nodeservice "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/grpc/node"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/grpc/tmservice"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	runtimeservices "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/runtime/services"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/api"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/config"
	servertypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/types"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/keeper"
	authsims "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/simulation"
	authtx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/vesting/types"
	authz "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz"
	authzkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz/module"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank"
	bankkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/capability/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/consensus/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis/types"
	distr "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution"
	distrkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/evidence/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/feegrant/module"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil"
	genutiltypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov"
	govclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/client"
	govkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params"
	paramsclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/client"
	paramskeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/types"
	paramproposal "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/types/proposal"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking"
	stakingkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	ica "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts"
	icacontroller "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/27-interchain-accounts/types"
	transfer "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/transfer"
	ibctransferkeeper "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/transfer/types"
	ibc "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core"
	ibcclient "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client"
	ibcclientclient "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/client"
	ibcclienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	porttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/05-port/types"
	ibcexported "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
	ibckeeper "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/keeper"
	solomachine "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/06-solomachine"
	ibctm "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/07-tendermint"
	ibcmock "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/mock"
	simappparams "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/simapp/params"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/simapp/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/mint"
)

const appName = "SimApp"

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
				upgradeclient.LegacyProposalHandler,
				upgradeclient.LegacyCancelProposalHandler,
				ibcclientclient.UpdateClientProposalHandler,
				ibcclientclient.UpgradeProposalHandler,
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		solomachine.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ibcmock.AppModuleBasic{},
		ica.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		vesting.AppModuleBasic{},
		consensus.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		mint.ModuleName:                {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		icatypes.ModuleName:            nil,
		ibcmock.ModuleName:             nil,
	}
)

var (
	_ App                     = (*SimApp)(nil)
	_ servertypes.Application = (*SimApp)(nil)
)

// SimApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type SimApp struct {
	*baseapp.BaseApp
	db                dbm.DB
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	ICAControllerKeeper   icacontrollerkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedIBCMockKeeper       capabilitykeeper.ScopedKeeper
	ScopedICAMockKeeper       capabilitykeeper.ScopedKeeper

	// make IBC modules public for test purposes
	// these modules are never directly routed to by the IBC Router
	ICAAuthModule ibcmock.IBCModule

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// the configurator
	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".simapp")
}

// NewSimApp returns a reference to an initialized SimApp.
func NewSimApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, skipUpgradeHeights map[int64]bool,
	homePath string, invCheckPeriod uint, encodingConfig simappparams.EncodingConfig,
	appOpts servertypes.AppOptions, baseAppOptions ...func(*baseapp.BaseApp),
) *SimApp {
	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(appName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey, crisistypes.StoreKey,
		distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey, upgradetypes.StoreKey, feegrant.StoreKey,
		evidencetypes.StoreKey, ibctransfertypes.StoreKey, icacontrollertypes.StoreKey, icahosttypes.StoreKey, capabilitytypes.StoreKey,
		authzkeeper.StoreKey, consensusparamtypes.StoreKey,
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	// NOTE: The ibcmock.MemStoreKey is just mounted for testing purposes. Actual applications should
	// not include this key.
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, ibcmock.MemStoreKey)

	app := &SimApp{
		BaseApp:           bApp,
		db:                db,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
		txConfig:          encodingConfig.TxConfig,
	}

	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, keys[consensusparamtypes.StoreKey], authtypes.NewModuleAddress(govtypes.ModuleName).String())
	bApp.SetParamStore(&app.ConsensusParamsKeeper)

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAControllerKeeper := app.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	scopedICAHostKeeper := app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	scopedIBCMockKeeper := app.CapabilityKeeper.ScopeToModule(ibcmock.ModuleName)
	scopedICAMockKeeper := app.CapabilityKeeper.ScopeToModule(ibcmock.ModuleName + icacontrollertypes.SubModuleName)

	// seal capability keeper after scoping modules
	app.CapabilityKeeper.Seal()

	// SDK module keepers

	app.AccountKeeper = authkeeper.NewAccountKeeper(appCodec, keys[authtypes.StoreKey], authtypes.ProtoBaseAccount, maccPerms, sdk.GetConfig().GetBech32AccountAddrPrefix(), authtypes.NewModuleAddress(govtypes.ModuleName).String())

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		app.AccountKeeper,
		BlockedAddresses(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey], app.AccountKeeper, app.BankKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.DistrKeeper = distrkeeper.NewKeeper(appCodec, keys[distrtypes.StoreKey], app.AccountKeeper, app.BankKeeper, app.StakingKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec, legacyAmino, keys[slashingtypes.StoreKey], app.StakingKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	app.CrisisKeeper = crisiskeeper.NewKeeper(appCodec, keys[crisistypes.StoreKey], invCheckPeriod,
		app.BankKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, keys[feegrant.StoreKey], app.AccountKeeper)
	app.UpgradeKeeper = upgradekeeper.NewKeeper(skipUpgradeHeights, keys[upgradetypes.StoreKey], appCodec, homePath, app.BaseApp, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	app.AuthzKeeper = authzkeeper.NewKeeper(keys[authzkeeper.StoreKey], appCodec, app.MsgServiceRouter(), app.AccountKeeper)

	// IBC Keepers

	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, keys[ibcexported.StoreKey], app.GetSubspace(ibcexported.ModuleName), app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper,
	)

	// register the proposal types
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(app.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper))

	govConfig := govtypes.DefaultConfig()
	/*
		Example of setting gov params:
		govConfig.MaxMetadataLen = 10000
	*/
	govKeeper := govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.AccountKeeper, app.BankKeeper,
		app.StakingKeeper, app.MsgServiceRouter(), govConfig, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Set legacy router for backwards compatibility with gov v1beta1
	govKeeper.SetLegacyRouter(govRouter)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(
		// register the governance hooks
		),
	)

	// ICA Controller keeper
	app.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec, keys[icacontrollertypes.StoreKey], app.GetSubspace(icacontrollertypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		scopedICAControllerKeeper, app.MsgServiceRouter(),
	)

	// ICA Host keeper
	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, keys[icahosttypes.StoreKey], app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper, app.AccountKeeper,
		scopedICAHostKeeper, app.MsgServiceRouter(),
	)
	app.ICAHostKeeper.WithQueryRouter(app.GRPCQueryRouter())

	// Create IBC Router
	ibcRouter := porttypes.NewRouter()

	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		app.AccountKeeper, app.BankKeeper, scopedTransferKeeper,
	)

	// Mock Module Stack

	// Mock Module setup for testing IBC and also acts as the interchain accounts authentication module
	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	mockModule := ibcmock.NewAppModule(&app.IBCKeeper.PortKeeper)

	// The mock module is used for testing IBC
	mockIBCModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(ibcmock.ModuleName, scopedIBCMockKeeper))
	ibcRouter.AddRoute(ibcmock.ModuleName, mockIBCModule)

	transferStack := transfer.NewIBCModule(app.TransferKeeper)

	// Add transfer stack to IBC Router
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)

	// Create Interchain Accounts Stack
	// initialize ICA module with mock module as the authentication module on the controller side
	var icaControllerStack porttypes.IBCModule
	icaControllerStack = ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp("", scopedICAMockKeeper))
	app.ICAAuthModule = icaControllerStack.(ibcmock.IBCModule)
	icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, app.ICAControllerKeeper)

	var icaHostStack porttypes.IBCModule = icahost.NewIBCModule(app.ICAHostKeeper)

	// Add host, controller & ica auth modules to IBC router
	ibcRouter.
		// the ICA Controller middleware needs to be explicitly added to the IBC Router because the
		// ICA controller module owns the port capability for ICA. The ICA authentication module
		// owns the channel capability.
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(ibcmock.ModuleName+icacontrollertypes.SubModuleName, icaControllerStack) // ica with mock auth module stack route to ica (top level of middleware stack)

	// Seal the IBC Router
	app.IBCKeeper.SetRouter(ibcRouter)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], app.StakingKeeper, app.SlashingKeeper,
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		// SDK app modules
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),

		// IBC modules
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
		mockModule,
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName, capabilitytypes.ModuleName, distrtypes.ModuleName, slashingtypes.ModuleName,
		evidencetypes.ModuleName, stakingtypes.ModuleName, ibcexported.ModuleName, ibctransfertypes.ModuleName, authtypes.ModuleName,
		banktypes.ModuleName, govtypes.ModuleName, crisistypes.ModuleName, genutiltypes.ModuleName, authz.ModuleName, feegrant.ModuleName,
		paramstypes.ModuleName, vestingtypes.ModuleName, icatypes.ModuleName, ibcmock.ModuleName, consensusparamtypes.ModuleName,
	)
	app.mm.SetOrderEndBlockers(
		crisistypes.ModuleName, govtypes.ModuleName, stakingtypes.ModuleName, ibcexported.ModuleName, ibctransfertypes.ModuleName,
		capabilitytypes.ModuleName, authtypes.ModuleName, banktypes.ModuleName, distrtypes.ModuleName, slashingtypes.ModuleName,
		genutiltypes.ModuleName, evidencetypes.ModuleName, authz.ModuleName, feegrant.ModuleName, paramstypes.ModuleName,
		upgradetypes.ModuleName, vestingtypes.ModuleName, icatypes.ModuleName, ibcmock.ModuleName, consensusparamtypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName, authtypes.ModuleName, banktypes.ModuleName, distrtypes.ModuleName, stakingtypes.ModuleName,
		slashingtypes.ModuleName, govtypes.ModuleName, crisistypes.ModuleName,
		ibcexported.ModuleName, genutiltypes.ModuleName, evidencetypes.ModuleName, authz.ModuleName, ibctransfertypes.ModuleName,
		icatypes.ModuleName, ibcmock.ModuleName, feegrant.ModuleName, paramstypes.ModuleName, upgradetypes.ModuleName,
		vestingtypes.ModuleName, consensusparamtypes.ModuleName,
	)

	app.mm.RegisterInvariants(app.CrisisKeeper)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// add test gRPC service for testing gRPC queries in isolation
	testdata.RegisterQueryServer(app.GRPCQueryRouter(), testdata.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	app.sm = module.NewSimulationManager(
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		params.NewAppModule(app.ParamsKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
	)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			HandlerOptions: ante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				SignModeHandler: encodingConfig.TxConfig.SignModeHandler(),
				FeegrantKeeper:  app.FeeGrantKeeper,
				SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
			},
			IBCKeeper: app.IBCKeeper,
		},
	)
	if err != nil {
		panic(err)
	}

	app.SetAnteHandler(anteHandler)

	app.SetEndBlocker(app.EndBlocker)

	app.setupUpgradeHandlers()
	app.setupUpgradeStoreLoaders()

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper
	app.ScopedICAControllerKeeper = scopedICAControllerKeeper
	app.ScopedICAHostKeeper = scopedICAHostKeeper

	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// note replicate if you do not need to test core IBC or light clients.
	app.ScopedIBCMockKeeper = scopedIBCMockKeeper
	app.ScopedICAMockKeeper = scopedICAMockKeeper

	return app
}

// Name returns the name of the App
func (app *SimApp) Name() string { return app.BaseApp.Name() }

// DB returns the backing database for test checkpointing.
func (app *SimApp) DB() dbm.DB { return app.db }

// BeginBlocker application updates every begin block
func (app *SimApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *SimApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *SimApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *SimApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *SimApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		// do not add the following modules to blocked addresses
		// this is only used for testing
		if acc == ibcmock.ModuleName {
			continue
		}

		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// GetModuleManager returns the app module manager
// NOTE: used for testing purposes
func (app *SimApp) GetModuleManager() *module.Manager {
	return app.mm
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *SimApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns SimApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *SimApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns SimApp's InterfaceRegistry
func (app *SimApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *SimApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *SimApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// TestingApp functions

// GetBaseApp implements the TestingApp interface.
func (app *SimApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetStakingKeeper implements the TestingApp interface.
func (app *SimApp) GetStakingKeeper() *stakingkeeper.Keeper {
	return app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *SimApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *SimApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// GetTxConfig implements the TestingApp interface.
func (app *SimApp) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// SimulationManager implements the SimulationApp interface
func (app *SimApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *SimApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register nodeservice grpc-gateway routes.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *SimApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *SimApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

func (app *SimApp) RegisterNodeService(clientCtx client.Context) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(ctx client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// ModuleAccountAddrsLegacy returns all the app's module account addresses.
func ModuleAccountAddrsLegacy() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range GetMaccPerms() {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

func BlockedAddresses() map[string]bool {
	modAccAddrs := ModuleAccountAddrsLegacy()

	// allow the following addresses to receive funds
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	delete(modAccAddrs, authtypes.NewModuleAddress(ibcmock.ModuleName).String())

	return modAccAddrs
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// TODO: ibc module subspaces can be removed after migration of params
	// https://github.com/cosmos/ibc-go/issues/2010
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)

	return paramsKeeper
}

// setupUpgradeHandlers sets all necessary upgrade handlers for testing purposes
func (app *SimApp) setupUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V5,
		upgrades.CreateDefaultUpgradeHandler(app.mm, app.configurator),
	)

	// NOTE: The moduleName arg of v6.CreateUpgradeHandler refers to the auth module ScopedKeeper name to which the channel capability should be migrated from.
	// This should be the same string value provided upon instantiation of the ScopedKeeper with app.CapabilityKeeper.ScopeToModule()
	// See: https://github.com/cosmos/ibc-go/blob/v6.1.0/testing/simapp/app.go#L310
	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V6,
		upgrades.CreateV6UpgradeHandler(
			app.mm,
			app.configurator,
			app.appCodec,
			app.keys[capabilitytypes.ModuleName],
			app.CapabilityKeeper,
			ibcmock.ModuleName+icacontrollertypes.SubModuleName,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V7,
		upgrades.CreateV7UpgradeHandler(
			app.mm,
			app.configurator,
			app.appCodec,
			app.IBCKeeper.ClientKeeper,
			app.ConsensusParamsKeeper,
			app.ParamsKeeper,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V7_1,
		upgrades.CreateV7LocalhostUpgradeHandler(app.mm, app.configurator, app.IBCKeeper.ClientKeeper),
	)
}

// setupUpgradeStoreLoaders sets all necessary store loaders required by upgrades.
func (app *SimApp) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		tmos.Exit(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if upgradeInfo.Name == upgrades.V7 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				consensusparamtypes.StoreKey,
				crisistypes.StoreKey,
			},
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
