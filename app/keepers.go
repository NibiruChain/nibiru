package app

import (
	"path/filepath"
	"strings"

	wasmdapp "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm"
	"github.com/cosmos/cosmos-sdk/client"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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
	govv1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
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
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
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
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/devgas/v1"
	devgaskeeper "github.com/NibiruChain/nibiru/v2/x/devgas/v1/keeper"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	"github.com/NibiruChain/nibiru/v2/x/epochs"
	epochskeeper "github.com/NibiruChain/nibiru/v2/x/epochs/keeper"
	epochstypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmmodule"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/genmsg"
	"github.com/NibiruChain/nibiru/v2/x/inflation"
	inflationkeeper "github.com/NibiruChain/nibiru/v2/x/inflation/keeper"
	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
	oracle "github.com/NibiruChain/nibiru/v2/x/oracle"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	"github.com/NibiruChain/nibiru/v2/x/sudo/keeper"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory"
	tokenfactorykeeper "github.com/NibiruChain/nibiru/v2/x/tokenfactory/keeper"
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
	crisisKeeper     crisiskeeper.Keeper
	upgradeKeeper    upgradekeeper.Keeper
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
	app.paramsKeeper = initParamsKeeper(
		app.appCodec, app.legacyAmino, app.keys[paramstypes.StoreKey], app.tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(app.appCodec, app.keys[consensusparamtypes.StoreKey], govModuleAddr)
	app.BaseApp.SetParamStore(&app.ConsensusParamsKeeper)

	/* Add capabilityKeeper and ScopeToModule for the ibc module
	   This allows authentication of object-capability permissions for each of
	   the IBC channels.
	*/
	app.capabilityKeeper = capabilitykeeper.NewKeeper(
		app.appCodec,
		app.keys[capabilitytypes.StoreKey],
		app.memKeys[capabilitytypes.MemStoreKey],
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

	nibiruBankKeeper := &evmkeeper.NibiruBankKeeper{
		BaseKeeper: bankkeeper.NewBaseKeeper(
			app.appCodec,
			app.keys[banktypes.StoreKey],
			app.AccountKeeper,
			BlockedAddresses(),
			govModuleAddr,
		),
		StateDB: nil,
	}
	app.BankKeeper = nibiruBankKeeper
	app.StakingKeeper = stakingkeeper.NewKeeper(
		app.appCodec,
		app.keys[stakingtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		govModuleAddr,
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		app.appCodec,
		app.keys[distrtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		govModuleAddr,
	)

	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	app.crisisKeeper = *crisiskeeper.NewKeeper(
		app.appCodec,
		app.keys[crisistypes.StoreKey],
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		govModuleAddr,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(app.appCodec, app.keys[feegrant.StoreKey], app.AccountKeeper)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))

	/*upgradeKeeper must be created before ibcKeeper. */
	app.upgradeKeeper = *upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		app.keys[upgradetypes.StoreKey],
		app.appCodec,
		homePath,
		app.BaseApp,
		govModuleAddr,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.slashingKeeper = slashingkeeper.NewKeeper(
		app.appCodec,
		app.legacyAmino,
		app.keys[slashingtypes.StoreKey],
		app.StakingKeeper,
		govModuleAddr,
	)

	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.slashingKeeper.Hooks()),
	)

	app.authzKeeper = authzkeeper.NewKeeper(
		app.keys[authzkeeper.StoreKey],
		app.appCodec,
		app.BaseApp.MsgServiceRouter(),
		app.AccountKeeper,
	)

	// ---------------------------------- Nibiru Chain x/ keepers

	app.SudoKeeper = keeper.NewKeeper(
		app.appCodec, app.keys[sudotypes.StoreKey],
	)

	app.OracleKeeper = oraclekeeper.NewKeeper(app.appCodec, app.keys[oracletypes.StoreKey],
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.StakingKeeper, app.slashingKeeper,
		app.SudoKeeper,
		distrtypes.ModuleName,
	)

	app.EpochsKeeper = epochskeeper.NewKeeper(
		app.appCodec, app.keys[epochstypes.StoreKey],
	)

	app.InflationKeeper = inflationkeeper.NewKeeper(
		app.appCodec, app.keys[inflationtypes.StoreKey], app.GetSubspace(inflationtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.StakingKeeper, app.SudoKeeper, authtypes.FeeCollectorName,
	)

	app.EpochsKeeper.SetHooks(
		epochstypes.NewMultiEpochHooks(
			app.InflationKeeper.Hooks(),
			app.OracleKeeper.Hooks(),
		),
	)

	evmKeeper := evmkeeper.NewKeeper(
		app.appCodec,
		app.keys[evm.StoreKey],
		app.tkeys[evm.TransientKey],
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		nibiruBankKeeper,
		app.StakingKeeper,
		cast.ToString(appOpts.Get("evm.tracer")),
	)
	app.EvmKeeper = &evmKeeper

	// ---------------------------------- IBC keepers

	app.ibcKeeper = ibckeeper.NewKeeper(
		app.appCodec,
		app.keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
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
		/* paramSubspace */ app.GetSubspace(ibctransfertypes.ModuleName),
		/* ibctransfertypes.ICS4Wrapper */ app.ibcFeeKeeper,
		/* ibctransfertypes.ChannelKeeper */ app.ibcKeeper.ChannelKeeper,
		/* ibctransfertypes.PortKeeper */ &app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
	)

	app.icaControllerKeeper = icacontrollerkeeper.NewKeeper(
		app.appCodec, app.keys[icacontrollertypes.StoreKey],
		app.GetSubspace(icacontrollertypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.ScopedICAControllerKeeper,
		app.MsgServiceRouter(),
	)

	app.icaHostKeeper = icahostkeeper.NewKeeper(
		app.appCodec,
		app.keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.ibcFeeKeeper,
		app.ibcKeeper.ChannelKeeper,
		&app.ibcKeeper.PortKeeper,
		app.AccountKeeper,
		app.ScopedICAHostKeeper,
		app.MsgServiceRouter(),
	)

	app.ScopedWasmKeeper = app.capabilityKeeper.ScopeToModule(wasmtypes.ModuleName)

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
		govModuleAddr,
	)

	// register the proposal types

	// Create evidence keeper.
	// This keeper automatically includes an evidence router.
	app.evidenceKeeper = *evidencekeeper.NewKeeper(
		app.appCodec, app.keys[evidencetypes.StoreKey], app.StakingKeeper,
		app.slashingKeeper,
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
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&app.upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.ibcKeeper.ClientKeeper))

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		app.appCodec,
		app.keys[govtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.MsgServiceRouter(),
		govConfig,
		govModuleAddr,
	)
	govKeeper.SetLegacyRouter(govRouter)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(),
	)
	app.EvmKeeper.AddPrecompiles(precompile.InitPrecompiles(app.AppKeepers.PublicKeepers))

	return wasmConfig
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

// registerNonDepinjectModules Load all the modules and stores them in the module manager
// NOTE: Any module instantiated in the module manager that is later modified
// must be passed by reference here.
func (app *NibiruApp) registerNonDepinjectModules(
	txConfig client.TxConfig,
	skipGenesisInvariants bool,
) {
	app.RegisterModules(
		// core modules
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx, txConfig),
		auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		bank.NewAppModule(app.appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(app.appCodec, *app.capabilityKeeper, false),
		feegrantmodule.NewAppModule(app.appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(app.appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		slashing.NewAppModule(app.appCodec, app.slashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(app.appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(app.appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
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
		wasm.NewAppModule(app.appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		devgas.NewAppModule(app.DevGasKeeper, app.AccountKeeper, app.GetSubspace(devgastypes.ModuleName)),
		tokenfactory.NewAppModule(app.TokenFactoryKeeper, app.AccountKeeper),

		crisis.NewAppModule(&app.crisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
	)

	app.ModuleManager.RegisterInvariants(&app.crisisKeeper)
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
