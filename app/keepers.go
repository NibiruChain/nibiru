package app

import (
	"path/filepath"
	"strings"

	wasmdapp "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	icacontroller "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	devgaskeeper "github.com/NibiruChain/nibiru/v2/x/devgas/v1/keeper"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	epochstypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	tokenfactorykeeper "github.com/NibiruChain/nibiru/v2/x/tokenfactory/keeper"
	tokenfactorytypes "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

const wasmVmContractMemoryLimit = 32

type AppKeepers struct {
	keepers.PublicKeepers

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

func (app *NibiruApp) initNonDepinjectKeepers(
	appOpts servertypes.AppOptions,
) (wasmConfig wasmtypes.WasmConfig) {
	govModuleAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	initSubspace(app.paramsKeeper)

	app.ScopedIBCKeeper = app.capabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	app.ScopedICAControllerKeeper = app.capabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	app.ScopedICAHostKeeper = app.capabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	// scopedFeeMockKeeper := app.capabilityKeeper.ScopeToModule(MockFeePort)
	app.ScopedTransferKeeper = app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

	// NOTE: the IBC mock keeper and application module is used only for testing core IBC. Do
	// not replicate if you do not need to test core IBC or light clients.
	_ = app.capabilityKeeper.ScopeToModule(ibcmock.ModuleName)

	app.ScopedWasmKeeper = app.capabilityKeeper.ScopeToModule(wasmtypes.ModuleName)

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

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))

	// ---------------------------------- Nibiru Chain x/ keepers
	app.EpochsKeeper.SetHooks(
		epochstypes.NewMultiEpochHooks(
			app.InflationKeeper.Hooks(),
			app.OracleKeeper.Hooks(),
		),
	)

	// ---------------------------------- IBC keepers

	app.ibcKeeper = ibckeeper.NewKeeper(
		app.appCodec,
		app.keys[ibcexported.StoreKey],
		app.getSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.upgradeKeeper,
		app.ScopedIBCKeeper,
	)

	// IBC Fee Module keeper
	app.ibcFeeKeeper = ibcfeekeeper.NewKeeper(
		app.appCodec, app.keys[ibcfeetypes.StoreKey],
		app.ibcKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	app.ibcTransferKeeper = ibctransferkeeper.NewKeeper(
		app.appCodec,
		app.keys[ibctransfertypes.StoreKey],
		/* paramSubspace */ app.getSubspace(ibctransfertypes.ModuleName),
		/* ibctransfertypes.ICS4Wrapper */ app.ibcFeeKeeper,
		/* ibctransfertypes.ChannelKeeper */ app.ibcKeeper.ChannelKeeper,
		/* ibctransfertypes.PortKeeper */ &app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
	)

	app.icaControllerKeeper = icacontrollerkeeper.NewKeeper(
		app.appCodec, app.keys[icacontrollertypes.StoreKey],
		app.getSubspace(icacontrollertypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.ScopedICAControllerKeeper,
		app.MsgServiceRouter(),
	)

	app.icaHostKeeper = icahostkeeper.NewKeeper(
		app.appCodec,
		app.keys[icahosttypes.StoreKey],
		app.getSubspace(icahosttypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.ScopedICAHostKeeper,
		app.MsgServiceRouter(),
	)

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
	// For example, if there are bindings for the x/inflation module, then the app
	// passed to GetWasmOpts must already have a non-nil InflationKeeper.
	supportedFeatures := strings.Join(wasmdapp.AllCapabilities(), ",")

	// Create wasm VM outside keeper so it can be re-used in client keeper
	wasmVM, err := wasmvm.NewVM(filepath.Join(wasmDir, "wasm"), supportedFeatures, wasmVmContractMemoryLimit, wasmConfig.ContractDebugMode, wasmConfig.MemoryCacheSize)
	if err != nil {
		panic(err)
	}

	wmha := wasmext.MsgHandlerArgs{
		Router:           app.MsgServiceRouter(),
		Ics4Wrapper:      app.ibcFeeKeeper,
		ChannelKeeper:    app.ibcKeeper.ChannelKeeper,
		CapabilityKeeper: app.ScopedWasmKeeper,
		BankKeeper:       app.BankKeeper,
		Unpacker:         app.appCodec,
		PortSource:       app.ibcTransferKeeper,
	}
	app.WasmMsgHandlerArgs = wmha
	app.WasmKeeper = wasmkeeper.NewKeeper(
		app.appCodec,
		app.keys[wasmtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		wmha.Ics4Wrapper, // ISC4 Wrapper: fee IBC middleware
		wmha.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		wmha.CapabilityKeeper,
		wmha.PortSource,
		wmha.Router,
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		supportedFeatures,
		govModuleAddr,
		append(GetWasmOpts(*app, appOpts, wmha), wasmkeeper.WithWasmEngine(wasmVM))...,
	)

	app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithVM(
		app.appCodec,
		app.keys[ibcwasmtypes.StoreKey],
		app.ibcKeeper.ClientKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		wasmVM,
		app.GRPCQueryRouter(),
	)

	// DevGas uses WasmKeeper
	app.DevGasKeeper = devgaskeeper.NewKeeper(
		app.keys[devgastypes.StoreKey],
		app.appCodec,
		app.BankKeeper,
		app.WasmKeeper,
		app.AccountKeeper,
		authtypes.FeeCollectorName,
		govModuleAddr,
	)

	app.TokenFactoryKeeper = tokenfactorykeeper.NewKeeper(
		app.keys[tokenfactorytypes.StoreKey],
		app.appCodec,
		app.BankKeeper,
		app.AccountKeeper,
		app.DistrKeeper,
		app.SudoKeeper,
		govModuleAddr,
	)

	// register the proposal types

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

	var wasmStack porttypes.IBCModule
	wasmStack = wasm.NewIBCHandler(app.WasmKeeper, app.ibcKeeper.ChannelKeeper, app.ibcFeeKeeper)
	wasmStack = ibcfee.NewIBCMiddleware(wasmStack, app.ibcFeeKeeper)

	// Add transfer stack to IBC Router
	ibcRouter.
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(wasmtypes.ModuleName, wasmStack)

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
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(app.upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.ibcKeeper.ClientKeeper))

	app.GovKeeper.SetLegacyRouter(govRouter)

	app.EvmKeeper.AddPrecompiles(precompile.InitPrecompiles(app.AppKeepers.PublicKeepers))

	return wasmConfig
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

func initSubspace(
	paramsKeeper paramskeeper.Keeper,
) paramskeeper.Keeper {
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)

	// ibc params keepers
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(ibcfeetypes.ModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	// wasm params keepers
	paramsKeeper.Subspace(wasmtypes.ModuleName)

	return paramsKeeper
}
