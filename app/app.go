package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	ibcmock "github.com/cosmos/ibc-go/v4/testing/mock"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
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
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
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
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
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
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
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
	ibcclientclient "github.com/cosmos/ibc-go/v4/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v4/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v4/testing"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/epochs"
	epochskeeper "github.com/NibiruChain/nibiru/x/epochs/keeper"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	oracle "github.com/NibiruChain/nibiru/x/oracle"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/perp"
	perpamm "github.com/NibiruChain/nibiru/x/perp/amm"
	perpammcli "github.com/NibiruChain/nibiru/x/perp/amm/cli"
	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/spot"
	spotkeeper "github.com/NibiruChain/nibiru/x/spot/keeper"
	spottypes "github.com/NibiruChain/nibiru/x/spot/types"
	"github.com/NibiruChain/nibiru/x/stablecoin"
	stablecoinkeeper "github.com/NibiruChain/nibiru/x/stablecoin/keeper"
	stablecointypes "github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/util"
	utiltypes "github.com/NibiruChain/nibiru/x/util/types"
)

const (
	AccountAddressPrefix = "nibi"
	appName              = "Nibiru"
	BondDenom            = "unibi"
	DisplayDenom         = "NIBI"
)

// IBC application testing ports
const (
	MockFeePort string = ibcmock.ModuleName + ibcfeetypes.ModuleName
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		BankModule{},
		capability.AppModuleBasic{},
		StakingModule{},
		MintModule{},
		distr.AppModuleBasic{},
		NewGovModuleBasic(
			paramsclient.ProposalHandler,
			distrclient.ProposalHandler,
			upgradeclient.ProposalHandler,
			upgradeclient.CancelProposalHandler,
			perpammcli.CreatePoolProposalHandler,
			perpammcli.EditPoolConfigProposalHandler,
			ibcclientclient.UpdateClientProposalHandler,
			ibcclientclient.UpgradeProposalHandler,
		),
		params.AppModuleBasic{},
		CrisisModule{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		vesting.AppModuleBasic{},
		// ibc 'AppModuleBasic's
		ibc.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
		// native x/
		spot.AppModuleBasic{},
		oracle.AppModuleBasic{},
		epochs.AppModuleBasic{},
		stablecoin.AppModuleBasic{},
		perp.AppModuleBasic{},
		perpamm.AppModuleBasic{},
		util.AppModule{},
		wasm.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:            nil,
		distrtypes.ModuleName:                 nil,
		minttypes.ModuleName:                  {authtypes.Minter},
		stakingtypes.BondedPoolName:           {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:        {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:                   {authtypes.Burner},
		spottypes.ModuleName:                  {authtypes.Minter, authtypes.Burner},
		oracletypes.ModuleName:                {},
		ibctransfertypes.ModuleName:           {authtypes.Minter, authtypes.Burner},
		ibcfeetypes.ModuleName:                nil,
		stablecointypes.ModuleName:            {authtypes.Minter, authtypes.Burner},
		perptypes.ModuleName:                  {authtypes.Minter, authtypes.Burner},
		perptypes.VaultModuleAccount:          {},
		perptypes.PerpEFModuleAccount:         {},
		perptypes.FeePoolModuleAccount:        {},
		epochstypes.ModuleName:                {},
		stablecointypes.StableEFModuleAccount: {authtypes.Burner},
		common.TreasuryPoolModuleAccount:      {},
		wasm.ModuleName:                       {},
	}
)

var (
	_ simapp.App              = (*NibiruApp)(nil)
	_ servertypes.Application = (*NibiruApp)(nil)
	_ ibctesting.TestingApp   = (*NibiruApp)(nil)
)

// NibiruApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type NibiruApp struct {
	*baseapp.BaseApp
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry codectypes.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*sdk.KVStoreKey
	tkeys   map[string]*sdk.TransientStoreKey
	memKeys map[string]*sdk.MemoryStoreKey

	// --------------------------------------------------------------------
	// NibiruTestApp Keepers
	// --------------------------------------------------------------------

	// AccountKeeper encodes/decodes accounts using the go-amino (binary) encoding/decoding library
	AccountKeeper authkeeper.AccountKeeper
	// BankKeeper defines a module interface that facilitates the transfer of coins between accounts
	BankKeeper       bankkeeper.Keeper
	capabilityKeeper *capabilitykeeper.Keeper
	stakingKeeper    stakingkeeper.Keeper
	slashingKeeper   slashingkeeper.Keeper
	mintKeeper       mintkeeper.Keeper
	/* distrKeeper is the keeper of the distribution store */
	distrKeeper    distrkeeper.Keeper
	govKeeper      govkeeper.Keeper
	crisisKeeper   crisiskeeper.Keeper
	upgradeKeeper  upgradekeeper.Keeper
	paramsKeeper   paramskeeper.Keeper
	authzKeeper    authzkeeper.Keeper
	FeeGrantKeeper feegrantkeeper.Keeper

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
	/* transferKeeper is for cross-chain fungible token transfers. */
	transferKeeper ibctransferkeeper.Keeper

	// make scoped keepers public for test purposes
	scopedIBCKeeper      capabilitykeeper.ScopedKeeper
	scopedTransferKeeper capabilitykeeper.ScopedKeeper

	// make IBC modules public for test purposes
	// these modules are never directly routed to by the IBC Router
	FeeMockModule ibcmock.IBCModule

	// ---------------
	// Nibiru keepers
	// ---------------
	EpochsKeeper     epochskeeper.Keeper
	PerpKeeper       perpkeeper.Keeper
	PerpAmmKeeper    perpammkeeper.Keeper
	SpotKeeper       spotkeeper.Keeper
	OracleKeeper     oraclekeeper.Keeper
	StablecoinKeeper stablecoinkeeper.Keeper

	// WASM keepers
	wasmKeeper       wasm.Keeper
	scopedWasmKeeper capabilitykeeper.ScopedKeeper

	// the module manager
	mm *module.Manager

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
func GetWasmOpts(appOpts servertypes.AppOptions) []wasm.Option {
	var wasmOpts []wasm.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}

	return wasmOpts
}

// NewNibiruApp returns a reference to an initialized NibiruApp.
func NewNibiruApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig simappparams.EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *NibiruApp {
	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(
		appName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		minttypes.StoreKey,
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
		perpammtypes.StoreKey,
		wasm.StoreKey,
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, stablecointypes.MemStoreKey)

	app := &NibiruApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

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
	app.scopedIBCKeeper = app.capabilityKeeper.ScopeToModule(ibchost.ModuleName)
	scopedFeeMockKeeper := app.capabilityKeeper.ScopeToModule(MockFeePort)
	app.scopedTransferKeeper = app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

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
	app.mintKeeper = mintkeeper.NewKeeper(
		appCodec, keys[minttypes.StoreKey], app.GetSubspace(minttypes.ModuleName), &stakingKeeper,
		app.AccountKeeper, app.BankKeeper, authtypes.FeeCollectorName,
	)
	app.distrKeeper = distrkeeper.NewKeeper(
		appCodec, keys[distrtypes.StoreKey], app.GetSubspace(distrtypes.ModuleName), app.AccountKeeper, app.BankKeeper,
		&stakingKeeper, authtypes.FeeCollectorName, app.ModuleAccountAddrs(),
	)
	app.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec, keys[slashingtypes.StoreKey], &stakingKeeper, app.GetSubspace(slashingtypes.ModuleName),
	)
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
		stakingtypes.NewMultiStakingHooks(app.distrKeeper.Hooks(), app.slashingKeeper.Hooks()),
	)

	app.authzKeeper = authzkeeper.NewKeeper(keys[authzkeeper.StoreKey], appCodec, app.BaseApp.MsgServiceRouter())

	// ---------------------------------- Nibiru Chain x/ keepers

	app.SpotKeeper = spotkeeper.NewKeeper(
		appCodec, keys[spottypes.StoreKey], app.GetSubspace(spottypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.distrKeeper)

	app.OracleKeeper = oraclekeeper.NewKeeper(appCodec, keys[oracletypes.StoreKey], app.GetSubspace(oracletypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.distrKeeper, app.stakingKeeper, distrtypes.ModuleName,
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

	app.EpochsKeeper.SetHooks(
		epochstypes.NewMultiEpochHooks(app.StablecoinKeeper.Hooks(), app.PerpKeeper.Hooks()),
	)

	// ---------------------------------- IBC keepers

	app.ibcKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibchost.StoreKey],
		app.GetSubspace(ibchost.ModuleName),
		app.stakingKeeper,
		app.upgradeKeeper,
		app.scopedIBCKeeper,
	)

	// IBC Fee Module keeper
	app.ibcFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, keys[ibcfeetypes.StoreKey], app.GetSubspace(ibcfeetypes.ModuleName),
		app.ibcKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper, app.AccountKeeper, app.BankKeeper,
	)

	scopedWasmKeeper := app.capabilityKeeper.ScopeToModule(wasm.ModuleName)

	wasmDir := filepath.Join(homePath, "data")
	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	supportedFeatures := "iterator,staking,stargate"
	app.wasmKeeper = wasm.NewKeeper(
		appCodec,
		keys[wasm.StoreKey],
		app.GetSubspace(wasm.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		app.distrKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		scopedWasmKeeper,
		app.transferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		supportedFeatures,
		GetWasmOpts(appOpts)...,
	)

	// register the proposal types

	govRouter := govtypes.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.distrKeeper)).
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
		app.scopedTransferKeeper,
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

	app.govKeeper = govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.GetSubspace(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, &app.stakingKeeper, govRouter,
	)

	// -------------------------- Module Options --------------------------

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	var skipGenesisInvariants = cast.ToBool(
		appOpts.Get(crisis.FlagSkipGenesisInvariants))

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
	utilModule := util.NewAppModule(app.BankKeeper)

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
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
		gov.NewAppModule(appCodec, app.govKeeper, app.AccountKeeper, app.BankKeeper),
		mint.NewAppModule(appCodec, app.mintKeeper, app.AccountKeeper),
		slashing.NewAppModule(appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
		distr.NewAppModule(appCodec, app.distrKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
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
		utilModule,

		// ibc
		evidence.NewAppModule(app.evidenceKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		ibctransfer.NewAppModule(app.transferKeeper),
		ibcfee.NewAppModule(app.ibcFeeKeeper),

		wasm.NewAppModule(appCodec, &app.wasmKeeper, app.stakingKeeper, app.AccountKeeper, app.BankKeeper),
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		stakingtypes.ModuleName,
		// native x/
		spottypes.ModuleName,
		oracletypes.ModuleName,
		epochstypes.ModuleName,
		stablecointypes.ModuleName,
		perpammtypes.ModuleName,
		perptypes.ModuleName,
		utiltypes.ModuleName,
		// ibc modules
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		wasm.ModuleName,
	)
	app.mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		// native x/
		epochstypes.ModuleName,
		stablecointypes.ModuleName,
		spottypes.ModuleName,
		oracletypes.ModuleName,
		perpammtypes.ModuleName,
		perptypes.ModuleName,
		utiltypes.ModuleName,
		// ibc
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		wasm.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		// native x/
		epochstypes.ModuleName,
		stablecointypes.ModuleName,
		spottypes.ModuleName,
		oracletypes.ModuleName,
		perpammtypes.ModuleName,
		perptypes.ModuleName,
		utiltypes.ModuleName,
		// ibc
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		wasm.ModuleName,
	)

	// Uncomment if you want to set a custom migration order here.
	// app.mm.SetOrderMigrations(custom order)

	app.mm.RegisterInvariants(&app.crisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)
	app.configurator = module.NewConfigurator(
		app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	app.upgradeKeeper.SetUpgradeHandler("v0.10.0", func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// no-op
		return fromVM, nil
	})

	// add test gRPC service for testing gRPC queries in isolation
	testdata.RegisterQueryServer(app.GRPCQueryRouter(), testdata.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	app.sm = module.NewSimulationManager(
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.govKeeper, app.AccountKeeper, app.BankKeeper),
		mint.NewAppModule(appCodec, app.mintKeeper, app.AccountKeeper),
		staking.NewAppModule(appCodec, app.stakingKeeper, app.AccountKeeper, app.BankKeeper),
		distr.NewAppModule(appCodec, app.distrKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper),
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

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	anteHandler, err := NewAnteHandler(AnteHandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   app.AccountKeeper,
			BankKeeper:      app.BankKeeper,
			FeegrantKeeper:  app.FeeGrantKeeper,
			SignModeHandler: encodingConfig.TxConfig.SignModeHandler(),
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
		IBCKeeper:         app.ibcKeeper,
		TxCounterStoreKey: keys[wasm.StoreKey],
		WasmConfig:        wasmConfig,
	})
	if err != nil {
		panic(fmt.Errorf("failed to create sdk.AnteHandler: %s", err))
	}

	app.SetAnteHandler(anteHandler)
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
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

	app.scopedWasmKeeper = scopedWasmKeeper

	return app
}

// Name returns the name of the App
func (app *NibiruApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *NibiruApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *NibiruApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *NibiruApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.upgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *NibiruApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
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
	subspace, _ := app.paramsKeeper.GetSubspace(moduleName)
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
	rpc.RegisterRoutes(clientCtx, apiSvr.Router)
	// Register legacy tx routes.
	authrest.RegisterTxRoutes(clientCtx, apiSvr.Router)
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterRESTRoutes(clientCtx, apiSvr.Router)
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
	tmservice.RegisterTendermintService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.interfaceRegistry)
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

func (app *NibiruApp) GetStakingKeeper() stakingkeeper.Keeper {
	return app.stakingKeeper
}

func (app *NibiruApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.ibcKeeper
}

func (app *NibiruApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.scopedIBCKeeper
}

/* EncodingConfig specifies the concrete encoding types to use for a given app.
   This is provided for compatibility between protobuf and amino implementations. */

func (app *NibiruApp) GetTxConfig() client.TxConfig {
	return MakeTestEncodingConfig().TxConfig
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

// initParamsKeeper init params perpammkeeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key,
	tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	// Native params keepers
	paramsKeeper.Subspace(spottypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)
	paramsKeeper.Subspace(epochstypes.ModuleName)
	paramsKeeper.Subspace(stablecointypes.ModuleName)
	paramsKeeper.Subspace(perptypes.ModuleName)
	// ibc params keepers
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	// wasm params keepers
	paramsKeeper.Subspace(wasm.ModuleName)

	return paramsKeeper
}
