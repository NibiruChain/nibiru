package app

import (
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	sigtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"

	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	"cosmossdk.io/store/types"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	wasmdapp "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
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
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

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

	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	// ---------------------------------------------------------------
	// IBC imports

	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v8/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
	"github.com/spf13/cast"

	// ---------------------------------------------------------------
	// Nibiru Custom Modules

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/devgas/v1"
	devgaskeeper "github.com/NibiruChain/nibiru/x/devgas/v1/keeper"
	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
	"github.com/NibiruChain/nibiru/x/epochs"
	epochskeeper "github.com/NibiruChain/nibiru/x/epochs/keeper"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmmodule"
	evmkeeper "github.com/NibiruChain/nibiru/x/evm/keeper"
	"github.com/NibiruChain/nibiru/x/genmsg"
	"github.com/NibiruChain/nibiru/x/inflation"
	inflationkeeper "github.com/NibiruChain/nibiru/x/inflation/keeper"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
	oracle "github.com/NibiruChain/nibiru/x/oracle"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"

	"github.com/NibiruChain/nibiru/x/sudo"
	"github.com/NibiruChain/nibiru/x/sudo/keeper"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"

	tokenfactory "github.com/NibiruChain/nibiru/x/tokenfactory"
	tokenfactorykeeper "github.com/NibiruChain/nibiru/x/tokenfactory/keeper"
	tokenfactorytypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

type AppKeepers struct {
	// AccountKeeper encodes/decodes accounts using the go-amino (binary) encoding/decoding library
	AccountKeeper authkeeper.AccountKeeper
	// BankKeeper defines a module interface that facilitates the transfer of coins between accounts
	BankKeeper       bankkeeper.Keeper
	capabilityKeeper *capabilitykeeper.Keeper
	stakingKeeper    *stakingkeeper.Keeper
	slashingKeeper   slashingkeeper.Keeper
	/* DistrKeeper is the keeper of the distribution store */
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	crisisKeeper          *crisiskeeper.Keeper
	upgradeKeeper         *upgradekeeper.Keeper
	paramsKeeper          paramskeeper.Keeper
	authzKeeper           authzkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

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

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper

	// make IBC modules public for test purposes
	// these modules are never directly routed to by the IBC Router
	FeeMockModule ibcmock.IBCModule

	// ---------------
	// Nibiru keepers
	// ---------------
	EpochsKeeper       epochskeeper.Keeper
	OracleKeeper       oraclekeeper.Keeper
	InflationKeeper    inflationkeeper.Keeper
	SudoKeeper         keeper.Keeper
	DevGasKeeper       devgaskeeper.Keeper
	TokenFactoryKeeper tokenfactorykeeper.Keeper
	EvmKeeper          evmkeeper.Keeper

	// WASM keepers
	WasmKeeper       wasmkeeper.Keeper
	ScopedWasmKeeper capabilitykeeper.ScopedKeeper
}

func initStoreKeys() (
	keys map[string]*types.KVStoreKey,
	tkeys map[string]*types.TransientStoreKey,
	memKeys map[string]*types.MemoryStoreKey,
) {
	keys = storetypes.NewKVStoreKeys(
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
		icahosttypes.StoreKey,
		icacontrollertypes.StoreKey,

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
	tkeys = storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys = storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	return keys, tkeys, memKeys
}

func (app *NibiruApp) InitKeepers(
	appOpts servertypes.AppOptions,
) (wasmConfig wasmtypes.WasmConfig) {
	appCodec := app.appCodec
	legacyAmino := app.legacyAmino
	bApp := app.BaseApp

	keys := app.keys
	tkeys := app.tkeys
	memKeys := app.memKeys

	govModuleAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	app.paramsKeeper = initParamsKeeper(
		appCodec, legacyAmino, keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		govModuleAddr,
		runtime.EventService{},
	)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

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
	app.ScopedICAControllerKeeper = app.capabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	app.ScopedICAHostKeeper = app.capabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	// scopedFeeMockKeeper := app.capabilityKeeper.ScopeToModule(MockFeePort)
	app.ScopedTransferKeeper = app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	_ = app.capabilityKeeper.ScopeToModule(ibcmock.ModuleName)

	// seal capability keeper after scoping modules
	// app.capabilityKeeper.Seal()

	// TODO: chore(upgrade): Potential breaking change on AccountKeeper dur
	// to ProtoBaseAccount replacement.
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(sdk.Bech32MainPrefix),
		sdk.Bech32MainPrefix,
		govModuleAddr,
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		BlockedAddresses(),
		govModuleAddr,
		app.Logger().With("module", "x/bank"),
	)

	// optional: enable sign mode textual by overwriting the default tx config (after setting the bank keeper)
	enabledSignModes := append(tx.DefaultSignModes, sigtypes.SignMode_SIGN_MODE_TEXTUAL)
	txConfigOpts := tx.ConfigOptions{
		EnabledSignModes:           enabledSignModes,
		TextualCoinMetadataQueryFn: txmodule.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	}
	txConfig, err := tx.NewTxConfigWithOptions(
		appCodec,
		txConfigOpts,
	)
	if err != nil {
		panic(err)
	}
	app.txConfig = txConfig

	app.stakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		govModuleAddr,
		authcodec.NewBech32Codec(sdk.Bech32PrefixValAddr),
		authcodec.NewBech32Codec(sdk.Bech32PrefixConsAddr),
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		authtypes.FeeCollectorName,
		govModuleAddr,
	)

	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	app.crisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[crisistypes.StoreKey]),
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		govModuleAddr,
		app.AccountKeeper.AddressCodec(),
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[feegrant.StoreKey]),
		app.AccountKeeper,
	)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))

	/*upgradeKeeper must be created before ibcKeeper. */
	app.upgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		app.BaseApp,
		govModuleAddr,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.stakingKeeper,
		govModuleAddr,
	)

	app.stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.slashingKeeper.Hooks()),
	)

	app.authzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[authzkeeper.StoreKey]),
		appCodec,
		app.BaseApp.MsgServiceRouter(),
		app.AccountKeeper,
	)

	// ---------------------------------- Nibiru Chain x/ keepers

	app.SudoKeeper = keeper.NewKeeper(
		appCodec, keys[sudotypes.StoreKey],
	)

	app.OracleKeeper = oraclekeeper.NewKeeper(appCodec, keys[oracletypes.StoreKey],
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.stakingKeeper, app.slashingKeeper,
		app.SudoKeeper,
		distrtypes.ModuleName,
	)

	app.EpochsKeeper = epochskeeper.NewKeeper(
		appCodec, keys[epochstypes.StoreKey],
	)

	app.InflationKeeper = inflationkeeper.NewKeeper(
		appCodec, keys[inflationtypes.StoreKey], app.GetSubspace(inflationtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.stakingKeeper, app.SudoKeeper, authtypes.FeeCollectorName,
	)

	app.EpochsKeeper.SetHooks(
		epochstypes.NewMultiEpochHooks(
			app.InflationKeeper.Hooks(),
			app.OracleKeeper.Hooks(),
		),
	)

	app.EvmKeeper = evmkeeper.NewKeeper(
		appCodec,
		keys[evm.StoreKey],
		tkeys[evm.TransientKey],
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
	)

	// ---------------------------------- IBC keepers

	app.ibcKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.stakingKeeper,
		app.upgradeKeeper,
		app.ScopedIBCKeeper,
		govModuleAddr,
	)

	// IBC Fee Module keeper
	app.ibcFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, keys[ibcfeetypes.StoreKey],
		app.ibcKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.ibcKeeper.ChannelKeeper,
		app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	app.ibcTransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		/* paramSubspace */ app.GetSubspace(ibctransfertypes.ModuleName),
		/* ibctransfertypes.ICS4Wrapper */ app.ibcFeeKeeper,
		/* ibctransfertypes.ChannelKeeper */ app.ibcKeeper.ChannelKeeper,
		/* ibctransfertypes.PortKeeper */ app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
		govModuleAddr,
	)

	app.icaControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec, keys[icacontrollertypes.StoreKey],
		app.GetSubspace(icacontrollertypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		app.ibcKeeper.PortKeeper,
		app.ScopedICAControllerKeeper,
		app.MsgServiceRouter(),
		govModuleAddr,
	)

	app.icaHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.ScopedICAHostKeeper,
		app.MsgServiceRouter(),
		govModuleAddr,
	)

	app.ScopedWasmKeeper = app.capabilityKeeper.ScopeToModule(wasmtypes.ModuleName)

	wasmDir := filepath.Join(homePath, "data")
	wasmConfig, err = wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	//
	// NOTE: This keeper depends on all of pointers to the the Keepers to which
	// it binds. Thus, it must be instantiated after those keepers have been
	// assigned.
	// For example, if there are bindings for the x/inflation module, then the app
	// passed to GetWasmOpts must already have a non-nil InflationKeeper.
	supportedFeatures := strings.Join(wasmdapp.AllCapabilities(), ",")
	app.WasmKeeper = wasmkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[wasmtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.ibcFeeKeeper, // ISC4 Wrapper: fee IBC middleware
		app.ibcKeeper.ChannelKeeper,
		app.ibcKeeper.PortKeeper,
		app.ScopedWasmKeeper,
		app.ibcTransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		supportedFeatures,
		govModuleAddr,
		GetWasmOpts(*app, appOpts)...,
	)

	// DevGas uses WasmKeeper
	app.DevGasKeeper = devgaskeeper.NewKeeper(
		keys[devgastypes.StoreKey],
		appCodec,
		app.BankKeeper,
		app.WasmKeeper,
		app.AccountKeeper,
		authtypes.FeeCollectorName,
		govModuleAddr,
	)

	app.TokenFactoryKeeper = tokenfactorykeeper.NewKeeper(
		keys[tokenfactorytypes.StoreKey],
		appCodec,
		app.BankKeeper,
		app.AccountKeeper,
		app.DistrKeeper,
		govModuleAddr,
	)

	// register the proposal types

	// Create evidence keeper.
	// This keeper automatically includes an evidence router.
	app.evidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		app.stakingKeeper,
		app.slashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)

	// Mock Module setup for testing IBC and also acts as the interchain accounts authentication module
	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	// mockModule := ibcmock.NewAppModule(&app.ibcKeeper.PortKeeper)

	// Create Transfer Stack
	// SendPacket, since it is originating from the application to core IBC:
	// transferKeeper.SendPacket -> fee.SendPacket -> channel.SendPacket

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
	// channel.RecvPacket -> fee.OnRecvPacket -> transfer.OnRecvPacket

	// transfer stack contains (from top to bottom):
	// - IBC Fee Middleware
	// - Transfer

	ibcRouter := porttypes.NewRouter()

	// create IBC module from bottom to top of stack
	var transferStack porttypes.IBCModule
	transferStack = ibctransfer.NewIBCModule(app.ibcTransferKeeper)
	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.ibcFeeKeeper)

	// Create Interchain Accounts Stack
	// SendPacket, since it is originating from the application to core IBC:
	// icaAuthModuleKeeper.SendTx -> icaController.SendPacket -> channel.SendPacket
	var icaControllerStack porttypes.IBCModule
	// integration point for custom authentication modules
	// see https://medium.com/the-interchain-foundation/ibc-go-v6-changes-to-interchain-accounts-and-how-it-impacts-your-chain-806c185300d7
	var noAuthzModule porttypes.IBCModule
	icaControllerStack = icacontroller.NewIBCMiddleware(noAuthzModule, app.icaControllerKeeper)

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket
	icaHostStack := icahost.NewIBCModule(app.icaHostKeeper)

	// Add transfer stack to IBC Router
	ibcRouter.
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack)

	// Create Mock IBC Fee module stack for testing
	// SendPacket, since it is originating from the application to core IBC:
	// mockModule.SendPacket -> fee.SendPacket -> channel.SendPacket

	// OnRecvPacket, message that originates from core IBC and goes down to app, the flow is the otherway
	// channel.RecvPacket -> fee.OnRecvPacket -> mockModule.OnRecvPacket

	// OnAcknowledgementPacket as this is where fee's are paid out
	// mockModule.OnAcknowledgementPacket -> fee.OnAcknowledgementPacket -> channel.OnAcknowledgementPacket

	// create fee wrapped mock module
	// feeMockModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewMockIBCApp(MockFeePort, scopedFeeMockKeeper))
	// app.FeeMockModule = feeMockModule
	// feeWithMockModule := ibcfee.NewIBCMiddleware(feeMockModule, app.ibcFeeKeeper)
	// ibcRouter.AddRoute(MockFeePort, feeWithMockModule)

	/* SetRouter finalizes all routes by sealing the router.
	   No more routes can be added. */
	app.ibcKeeper.SetRouter(ibcRouter)

	govRouter := govv1beta1types.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govv1beta1types.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		// AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&app.upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.ibcKeeper.ClientKeeper))

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.stakingKeeper,
		app.DistrKeeper,
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

func (app *NibiruApp) initAppModules(
	encodingConfig EncodingConfig,
	skipGenesisInvariants bool,
	txConfig client.TxConfig,
) []module.AppModule {
	appCodec := app.appCodec

	return []module.AppModule{
		// core modules
		genutil.NewAppModule(
			app.AccountKeeper, app.stakingKeeper, app,
			txConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.capabilityKeeper, false),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.stakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.stakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.upgradeKeeper, app.AccountKeeper.AddressCodec()),
		params.NewAppModule(app.paramsKeeper),
		authzmodule.NewAppModule(appCodec, app.authzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),

		// Nibiru modules
		oracle.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper),
		epochs.NewAppModule(appCodec, app.EpochsKeeper),
		inflation.NewAppModule(app.InflationKeeper, app.AccountKeeper, *app.stakingKeeper),
		sudo.NewAppModule(appCodec, app.SudoKeeper),
		genmsg.NewAppModule(app.MsgServiceRouter()),

		// ibc
		evidence.NewAppModule(app.evidenceKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		ibctransfer.NewAppModule(app.ibcTransferKeeper),
		ibcfee.NewAppModule(app.ibcFeeKeeper),
		ica.NewAppModule(&app.icaControllerKeeper, &app.icaHostKeeper),

		evmmodule.NewAppModule(&app.EvmKeeper, app.AccountKeeper),

		// wasm
		wasm.NewAppModule(
			appCodec, &app.WasmKeeper, app.stakingKeeper, app.AccountKeeper,
			app.BankKeeper, app.MsgServiceRouter(),
			app.GetSubspace(wasmtypes.ModuleName)),
		devgas.NewAppModule(
			app.DevGasKeeper, app.AccountKeeper,
			app.GetSubspace(devgastypes.ModuleName)),
		tokenfactory.NewAppModule(
			app.TokenFactoryKeeper, app.AccountKeeper,
		),

		crisis.NewAppModule(app.crisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
	}
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
		vestingtypes.ModuleName,

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

// initModuleManager Load all the modules and stores them in the module manager
// NOTE: Any module instantiated in the module manager that is later modified
// must be passed by reference here.
func (app *NibiruApp) initModuleManager(
	encodingConfig EncodingConfig,
	skipGenesisInvariants bool,
) {
	app.ModuleManager = module.NewManager(
		app.initAppModules(encodingConfig, skipGenesisInvariants, app.txConfig)...,
	)

	orderedModules := orderedModuleNames()
	app.ModuleManager.SetOrderBeginBlockers(orderedModules...)
	app.ModuleManager.SetOrderEndBlockers(orderedModules...)
	app.ModuleManager.SetOrderInitGenesis(orderedModules...)
	app.ModuleManager.SetOrderExportGenesis(orderedModules...)

	// Uncomment if you want to set a custom migration order here.
	// app.mm.SetOrderMigrations(custom order)

	app.ModuleManager.RegisterInvariants(app.crisisKeeper)
	app.configurator = module.NewConfigurator(
		app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.ModuleManager.RegisterServices(app.configurator)

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
			// upgradeclient.LegacyProposalHandler,
			// upgradeclient.LegacyCancelProposalHandler,
			// seems to be removed per https://github.com/cosmos/ibc-go/pull/4620
			// ibcclientclient.UpdateClientProposalHandler,
			// ibcclientclient.UpgradeProposalHandler,
		),
		params.AppModuleBasic{},
		CrisisModule{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		groupmodule.AppModuleBasic{},
		vesting.AppModuleBasic{},
		// ibc 'AppModuleBasic's
		ibc.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		ica.AppModuleBasic{},
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
