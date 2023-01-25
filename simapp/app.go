package simapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CosmWasm/wasmd/x/wasm"

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
	ibctransfer "github.com/cosmos/ibc-go/v3/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v3/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v3/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v3/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v3/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v3/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	nibiapp "github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/dex"
	dexkeeper "github.com/NibiruChain/nibiru/x/dex/keeper"
	dextypes "github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/epochs"
	epochskeeper "github.com/NibiruChain/nibiru/x/epochs/keeper"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/oracle"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/perp"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/stablecoin"
	stablecoinkeeper "github.com/NibiruChain/nibiru/x/stablecoin/keeper"
	stablecointypes "github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/vpool"
	vpoolcli "github.com/NibiruChain/nibiru/x/vpool/client/cli"
	vpoolkeeper "github.com/NibiruChain/nibiru/x/vpool/keeper"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

const (
	Name      = "nibiru"
	AppName   = "Nibiru"
	BondDenom = "unibi"
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
		nibiapp.BankModule{},
		capability.AppModuleBasic{},
		nibiapp.StakingModule{},
		nibiapp.MintModule{},
		distr.AppModuleBasic{},
		nibiapp.NewGovModuleBasic(
			paramsclient.ProposalHandler,
			distrclient.ProposalHandler,
			upgradeclient.ProposalHandler,
			upgradeclient.CancelProposalHandler,
			vpoolcli.CreatePoolProposalHandler,
			vpoolcli.EditPoolConfigProposalHandler,
			// pricefeedcli.RemoveOracleProposalHandler, // TODO
			ibcclientclient.UpdateClientProposalHandler,
			ibcclientclient.UpgradeProposalHandler,
		),
		params.AppModuleBasic{},
		nibiapp.CrisisModule{},
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
		oracle.AppModuleBasic{},
		dex.AppModuleBasic{},
		epochs.AppModuleBasic{},
		stablecoin.AppModuleBasic{},
		perp.AppModuleBasic{},
		vpool.AppModuleBasic{},
		wasm.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:            nil,
		distrtypes.ModuleName:                 nil,
		minttypes.ModuleName:                  {authtypes.Minter},
		stakingtypes.BondedPoolName:           {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:        {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:                   {authtypes.Burner},
		dextypes.ModuleName:                   {authtypes.Minter, authtypes.Burner},
		ibctransfertypes.ModuleName:           {authtypes.Minter, authtypes.Burner},
		stablecointypes.ModuleName:            {authtypes.Minter, authtypes.Burner},
		perptypes.ModuleName:                  {authtypes.Minter, authtypes.Burner},
		perptypes.VaultModuleAccount:          {},
		perptypes.PerpEFModuleAccount:         {},
		perptypes.FeePoolModuleAccount:        {},
		epochstypes.ModuleName:                {},
		oracletypes.ModuleName:                {},
		stablecointypes.StableEFModuleAccount: {authtypes.Burner},
		common.TreasuryPoolModuleAccount:      {},
		wasm.ModuleName:                       {},
	}
)

var (
	_ simapp.App              = (*NibiruTestApp)(nil)
	_ servertypes.Application = (*NibiruTestApp)(nil)
	_ ibctesting.TestingApp   = (*NibiruTestApp)(nil)
)

// NibiruTestApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type NibiruTestApp struct {
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
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	MintKeeper       mintkeeper.Keeper
	/* DistrKeeper is the keeper of the distribution store */
	DistrKeeper    distrkeeper.Keeper
	GovKeeper      govkeeper.Keeper
	CrisisKeeper   crisiskeeper.Keeper
	UpgradeKeeper  upgradekeeper.Keeper
	ParamsKeeper   paramskeeper.Keeper
	AuthzKeeper    authzkeeper.Keeper
	FeeGrantKeeper feegrantkeeper.Keeper

	// --------------------------------------------------------------------
	// IBC keepers
	// --------------------------------------------------------------------
	/* EvidenceKeeper is responsible for managing persistence, state transitions
	   and query handling for the evidence module. It is required to set up
	   the IBC light client misbehavior evidence route. */
	EvidenceKeeper evidencekeeper.Keeper
	/* IBCKeeper defines each ICS keeper for IBC. IBCKeeper must be a pointer in
	   the nibiapp, so we can SetRouter on it correctly. */
	IBCKeeper *ibckeeper.Keeper
	/* TransferKeeper is for cross-chain fungible token transfers. */
	TransferKeeper ibctransferkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// ---------------
	// Nibiru keepers
	// ---------------
	OracleKeeper     oraclekeeper.Keeper
	DexKeeper        dexkeeper.Keeper
	StablecoinKeeper stablecoinkeeper.Keeper
	PerpKeeper       perpkeeper.Keeper
	EpochsKeeper     epochskeeper.Keeper
	VpoolKeeper      vpoolkeeper.Keeper

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

// NewNibiruTestApp returns a reference to an initialized NibiruTestApp.
func NewNibiruTestApp(
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
) *NibiruTestApp {
	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(AppName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
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
		// nibiru x/ keys
		oracletypes.StoreKey,
		dextypes.StoreKey,
		stablecointypes.StoreKey,
		epochstypes.StoreKey,
		perptypes.StoreKey,
		vpooltypes.StoreKey,
		wasm.StoreKey,
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	// NOTE: The testingkey is just mounted for testing purposes. Actual applications should
	// not include this key.
	memKeys := sdk.NewMemoryStoreKeys(
		capabilitytypes.MemStoreKey, "testingkey",
		stablecointypes.MemStoreKey,
	)

	app := &NibiruTestApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	app.ParamsKeeper = initParamsKeeper(
		appCodec, legacyAmino, keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	bApp.SetParamStore(app.ParamsKeeper.Subspace(baseapp.Paramspace).
		WithKeyTable(paramskeeper.ConsensusParamsKeyTable()))

	/* Add CapabilityKeeper and ScopeToModule for the ibc module
	   This allows authentication of object-capability permissions for each of
	   the IBC channels.
	*/
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	app.ScopedIBCKeeper = app.CapabilityKeeper.ScopeToModule(ibchost.ModuleName)
	app.ScopedTransferKeeper = app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

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
	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec, keys[minttypes.StoreKey], app.GetSubspace(minttypes.ModuleName), &stakingKeeper,
		app.AccountKeeper, app.BankKeeper, authtypes.FeeCollectorName,
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec, keys[distrtypes.StoreKey], app.GetSubspace(distrtypes.ModuleName), app.AccountKeeper, app.BankKeeper,
		&stakingKeeper, authtypes.FeeCollectorName, app.ModuleAccountAddrs(),
	)
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec, keys[slashingtypes.StoreKey], &stakingKeeper, app.GetSubspace(slashingtypes.ModuleName),
	)
	app.CrisisKeeper = crisiskeeper.NewKeeper(
		app.GetSubspace(crisistypes.ModuleName), invCheckPeriod, app.BankKeeper, authtypes.FeeCollectorName,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, keys[feegrant.StoreKey], app.AccountKeeper)

	/*UpgradeKeeper must be created before IBCKeeper. */
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		app.BaseApp)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper = *stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	app.AuthzKeeper = authzkeeper.NewKeeper(keys[authzkeeper.StoreKey], appCodec, app.BaseApp.MsgServiceRouter())

	// ---------------------------------- Nibiru Chain x/ keepers

	app.OracleKeeper = oraclekeeper.NewKeeper(
		appCodec,
		keys[oracletypes.StoreKey],
		app.GetSubspace(oracletypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, &stakingKeeper,
		distrtypes.ModuleName,
	)

	app.DexKeeper = dexkeeper.NewKeeper(
		appCodec, keys[dextypes.StoreKey], app.GetSubspace(dextypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper)

	app.StablecoinKeeper = stablecoinkeeper.NewKeeper(
		appCodec, keys[stablecointypes.StoreKey], memKeys[stablecointypes.MemStoreKey],
		app.GetSubspace(stablecointypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.OracleKeeper, app.DexKeeper,
	)

	app.VpoolKeeper = vpoolkeeper.NewKeeper(
		appCodec,
		keys[vpooltypes.StoreKey],
		app.OracleKeeper,
	)

	app.EpochsKeeper = epochskeeper.NewKeeper(
		appCodec, keys[epochstypes.StoreKey],
	)

	app.PerpKeeper = perpkeeper.NewKeeper(
		appCodec, keys[perptypes.StoreKey],
		app.GetSubspace(perptypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.OracleKeeper, app.VpoolKeeper, app.EpochsKeeper,
	)

	app.EpochsKeeper.SetHooks(
		epochstypes.NewMultiEpochHooks(app.StablecoinKeeper.Hooks(), app.PerpKeeper.Hooks()),
	)

	// ---------------------------------- IBC keepers

	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibchost.StoreKey],
		app.GetSubspace(ibchost.ModuleName),
		app.StakingKeeper,
		app.UpgradeKeeper,
		app.ScopedIBCKeeper,
	)

	scopedWasmKeeper := app.CapabilityKeeper.ScopeToModule(wasm.ModuleName)

	wasmDir := filepath.Join(homePath, "data")
	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	supportedFeatures := "iterator,staking,stargate"
	var wasmOpts []wasm.Option
	app.wasmKeeper = wasm.NewKeeper(
		appCodec,
		keys[wasm.StoreKey],
		app.GetSubspace(wasm.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		scopedWasmKeeper,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		supportedFeatures,
		wasmOpts...,
	)

	// register the proposal types

	govRouter := govtypes.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.DistrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(app.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper)).
		AddRoute(vpooltypes.RouterKey, vpool.NewVpoolProposalHandler(app.VpoolKeeper))

	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		/* paramSubspace */ app.GetSubspace(ibctransfertypes.ModuleName),
		/* ibctransfertypes.ICS4Wrapper */ app.IBCKeeper.ChannelKeeper,
		/* ibctransfertypes.ChannelKeeper */ app.IBCKeeper.ChannelKeeper,
		/* ibctransfertypes.PortKeeper */ &app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
	)
	transferModule := ibctransfer.NewAppModule(app.TransferKeeper)
	transferIBCModule := ibctransfer.NewIBCModule(app.TransferKeeper)

	// Create evidence keeper.
	// This keeper automatically includes an evidence router.
	app.EvidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], &app.StakingKeeper,
		app.SlashingKeeper,
	)

	/* Create IBC module and a static IBC router */
	ibcRouter := porttypes.NewRouter()
	/* Add an ibc-transfer module route, then set it and seal it. */
	ibcRouter.AddRoute(
		/* module    */ ibctransfertypes.ModuleName,
		/* ibcModule */ transferIBCModule).
		AddRoute(wasm.ModuleName, wasm.NewIBCHandler(app.wasmKeeper, app.IBCKeeper.ChannelKeeper))

	/* SetRouter finalizes all routes by sealing the router.
	   No more routes can be added. */
	app.IBCKeeper.SetRouter(ibcRouter)

	app.GovKeeper = govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.GetSubspace(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, &app.StakingKeeper, govRouter,
	)

	// -------------------------- Module Options --------------------------

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	var skipGenesisInvariants = cast.ToBool(
		appOpts.Get(crisis.FlagSkipGenesisInvariants))

	oracleModule := oracle.NewAppModule(
		appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper)

	dexModule := dex.NewAppModule(
		appCodec, app.DexKeeper, app.AccountKeeper, app.BankKeeper)
	epochsModule := epochs.NewAppModule(appCodec, app.EpochsKeeper)
	stablecoinModule := stablecoin.NewAppModule(
		appCodec, app.StablecoinKeeper, app.AccountKeeper, app.BankKeeper,
		nil,
	)
	perpModule := perp.NewAppModule(
		appCodec, app.PerpKeeper, app.AccountKeeper, app.BankKeeper,
		nil,
	)
	vpoolModule := vpool.NewAppModule(
		appCodec, app.VpoolKeeper, nil,
	)

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper),
		crisis.NewAppModule(&app.CrisisKeeper, skipGenesisInvariants),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		upgrade.NewAppModule(app.UpgradeKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		// native x/
		oracleModule,
		dexModule,
		stablecoinModule,
		epochsModule,
		vpoolModule,
		perpModule,
		// ibc
		evidence.NewAppModule(app.EvidenceKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		transferModule,
		wasm.NewAppModule(appCodec, &app.wasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
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
		dextypes.ModuleName,
		epochstypes.ModuleName,
		stablecointypes.ModuleName,
		vpooltypes.ModuleName,
		perptypes.ModuleName,
		oracletypes.ModuleName,
		// ibc modules
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		wasm.ModuleName,
	)
	app.mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		oracletypes.ModuleName,
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
		dextypes.ModuleName,
		epochstypes.ModuleName,
		stablecointypes.ModuleName,
		vpooltypes.ModuleName,
		perptypes.ModuleName,
		// ibc
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
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
		oracletypes.ModuleName,
		dextypes.ModuleName,
		epochstypes.ModuleName,
		stablecointypes.ModuleName,
		vpooltypes.ModuleName,
		perptypes.ModuleName,
		// ibc
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		wasm.ModuleName,
	)

	// Uncomment if you want to set a custom migration order here.
	// nibiapp.mm.SetOrderMigrations(custom order)

	app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)
	app.configurator = module.NewConfigurator(
		app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	app.UpgradeKeeper.SetUpgradeHandler("v0.10.0", func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
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
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		// native x/
		epochsModule,
		oracleModule,
		perpModule,
		vpoolModule,
		dexModule,
		// ibc
		capability.NewAppModule(appCodec, *app.CapabilityKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		transferModule,
	)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	anteHandler, err := nibiapp.NewAnteHandler(nibiapp.AnteHandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   app.AccountKeeper,
			BankKeeper:      app.BankKeeper,
			FeegrantKeeper:  app.FeeGrantKeeper,
			SignModeHandler: encodingConfig.TxConfig.SignModeHandler(),
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
		IBCKeeper:         app.IBCKeeper,
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
		`CapabilityKeeper.ScopeToModule`.


		Calling 'nibiapp.CapabilityKeeper.Seal()' initializes and seals the capability
		keeper such that all persistent capabilities are loaded in-memory and prevent
		any further modules from creating scoped sub-keepers.

		NOTE: This must be done during creation of baseapp rather than in InitChain so
		that in-memory capabilities get regenerated on nibiapp restart.
		Note that since this reads from the store, we can only perform the seal
		when `loadLatest` is set to true.
		*/
		app.CapabilityKeeper.Seal()
	}

	app.scopedWasmKeeper = scopedWasmKeeper

	return app
}

// Name returns the name of the App
func (app *NibiruTestApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *NibiruTestApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *NibiruTestApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *NibiruTestApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState nibiapp.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *NibiruTestApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the nibiapp's module account addresses.
func (app *NibiruTestApp) ModuleAccountAddrs() map[string]bool {
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
func (app *NibiruTestApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns SimApp's nibiapp codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *NibiruTestApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns App's InterfaceRegistry
func (app *NibiruTestApp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruTestApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruTestApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *NibiruTestApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *NibiruTestApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *NibiruTestApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *NibiruTestApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
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
func (app *NibiruTestApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(
		app.BaseApp.GRPCQueryRouter(), clientCtx,
		app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *NibiruTestApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.interfaceRegistry)
}

// ------------------------------------------------------------------------
// Functions for ibc-go TestingApp
// ------------------------------------------------------------------------

/* GetBaseApp, GetStakingKeeper, GetIBCKeeper, and GetScopedIBCKeeper are part
   of the implementation of the TestingApp interface
*/

func (app *NibiruTestApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

func (app *NibiruTestApp) GetStakingKeeper() stakingkeeper.Keeper {
	return app.StakingKeeper
}

func (app *NibiruTestApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

func (app *NibiruTestApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

/* EncodingConfig specifies the concrete encoding types to use for a given nibiapp.
   This is provided for compatibility between protobuf and amino implementations. */

func (app *NibiruTestApp) GetTxConfig() client.TxConfig {
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

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// initParamsKeeper init params vpoolkeeper and its subspaces
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
	// Native module params keepers
	paramsKeeper.Subspace(dextypes.ModuleName)
	paramsKeeper.Subspace(epochstypes.ModuleName)
	paramsKeeper.Subspace(stablecointypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)
	paramsKeeper.Subspace(perptypes.ModuleName)
	// ibc params keepers
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	// wasm params keepers
	paramsKeeper.Subspace(wasm.ModuleName)

	return paramsKeeper
}
