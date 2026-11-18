package upgrade

import (
	"context"
	"encoding/json"
	"fmt"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"

	modulev1 "cosmossdk.io/api/cosmos/upgrade/module/v1"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/baseapp"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/runtime"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	servertypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/types"
	store "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/upgrade/client/cli"
	"github.com/NibiruChain/nibiru/v2/x/upgrade/keeper"
)

func init() {
	types.RegisterLegacyAminoCodec(codec.NewLegacyAmino())
}

const (
	consensusVersion uint64 = 3
)

var (
	_ module.BeginBlockAppModule = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
)

// AppModuleBasic implements the sdk.AppModuleBasic interface
type AppModuleBasic struct{}

// Name returns the ModuleName
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the upgrade types on the LegacyAmino codec
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the upgrade module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetQueryCmd returns the CLI query commands for this module
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// GetTxCmd returns the CLI transaction commands for this module
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// RegisterInterfaces registers interfaces and implementations of the upgrade module.
func (b AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// AppModule implements the sdk.AppModule interface
type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
	}
}

var _ appmodule.AppModule = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// RegisterInvariants does nothing, there are no invariants to enforce
func (AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	if err := cfg.RegisterMigration(types.ModuleName, 2, func(sdk.Context) error { return nil }); err != nil {
		panic(err)
	}
	m := keeper.NewMigrator(am.keeper)
	err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2)
	if err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", types.ModuleName, err))
	}
}

// InitGenesis restores exceptional host failure metadata. Contract state is
// imported by Wasm genesis; the app activates it after all module genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if err := am.keeper.InitSelectionFailureGenesis(ctx, data); err != nil {
		panic(err)
	}
	// set version map automatically if available
	if versionMap := am.keeper.GetInitVersionMap(); versionMap != nil {
		// chains can still use a custom init chainer for setting the version map
		// this means that we need to combine the manually wired modules version map with app wiring enabled modules version map
		for name, version := range am.keeper.GetModuleVersionMap(ctx) {
			if _, ok := versionMap[name]; !ok {
				versionMap[name] = version
			}
		}

		am.keeper.SetModuleVersionMap(ctx, versionMap)
	}

	return []abci.ValidatorUpdate{}
}

// DefaultGenesis is an empty object
func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	return []byte("{}")
}

// ValidateGenesis checks the exceptional native failure receipt.
func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, config client.TxEncodingConfig, data json.RawMessage) error {
	return keeper.ValidateSelectionFailureGenesis(data)
}

// ExportGenesis preserves native failure suppression; Wasm exports contract state.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return am.keeper.SelectionFailureGenesis(ctx)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return consensusVersion }

// BeginBlock calls the upgrade module hooks
//
// CONTRACT: this is registered in BeginBlocker *before* all other modules' BeginBlock functions
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	BeginBlocker(am.keeper, ctx, req)
}

//
// App Wiring Setup
//

func init() {
	appmodule.Register(&modulev1.Module{},
		appmodule.Provide(ProvideModule),
		appmodule.Invoke(PopulateVersionMap),
	)
}

type UpgradeInputs struct {
	depinject.In

	Config *modulev1.Module
	Key    *store.KVStoreKey
	Cdc    codec.Codec

	AppOpts servertypes.AppOptions `optional:"true"`
}

type UpgradeOutputs struct {
	depinject.Out

	UpgradeKeeper *keeper.Keeper
	Module        appmodule.AppModule
	GovHandler    govv1beta1.HandlerRoute
	BaseAppOption runtime.BaseAppOption
}

func ProvideModule(in UpgradeInputs) UpgradeOutputs {
	var (
		homePath           string
		skipUpgradeHeights = make(map[int64]bool)
	)

	if in.AppOpts != nil {
		for _, h := range cast.ToIntSlice(in.AppOpts.Get(server.FlagUnsafeSkipUpgrades)) {
			skipUpgradeHeights[int64(h)] = true
		}

		homePath = cast.ToString(in.AppOpts.Get(flags.FlagHome))
	}

	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	// set the governance module account as the authority for conducting upgrades
	k := keeper.NewKeeper(skipUpgradeHeights, in.Key, in.Cdc, homePath, nil, authority.String())
	baseappOpt := func(app *baseapp.BaseApp) {
		k.SetVersionSetter(app)
	}
	m := NewAppModule(k)
	gh := govv1beta1.HandlerRoute{RouteKey: types.RouterKey, Handler: NewSoftwareUpgradeProposalHandler(k)}

	return UpgradeOutputs{UpgradeKeeper: k, Module: m, GovHandler: gh, BaseAppOption: baseappOpt}
}

func PopulateVersionMap(upgradeKeeper *keeper.Keeper, modules map[string]appmodule.AppModule) {
	if upgradeKeeper == nil {
		return
	}

	upgradeKeeper.SetInitVersionMap(module.NewManagerFromMap(modules).GetVersionMap())
}
