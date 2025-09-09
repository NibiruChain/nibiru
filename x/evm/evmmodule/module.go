// Copyright (c) 2023-2024 Nibi, Inc.
package evmmodule

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/exported"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/cli"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"

	bankmodulev1 "cosmossdk.io/api/cosmos/bank/module/v1"

	modulev1 "github.com/NibiruChain/nibiru/v2/api/eth/evm/module"
)

// consensusVersion: EVM module consensus version for upgrades.
const consensusVersion = 1

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.EndBlockAppModule   = AppModule{}
	_ module.BeginBlockAppModule = AppModule{}
)

// AppModuleBasic defines the basic application module used by the evm module.
type AppModuleBasic struct{}

// Name returns the evm module's name.
func (AppModuleBasic) Name() string {
	return evm.ModuleName
}

// RegisterLegacyAminoCodec registers the module's types with the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	evm.RegisterLegacyAminoCodec(cdc)
}

// ConsensusVersion returns the consensus state-breaking version for the module.
func (AppModuleBasic) ConsensusVersion() uint64 {
	return consensusVersion
}

// DefaultGenesis returns default genesis state as raw bytes for the evm
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(evm.DefaultGenesisState())
}

// ValidateGenesis is the validation check of the Genesis
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genesisState evm.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genesisState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", evm.ModuleName, err)
	}

	return genesisState.Validate()
}

// RegisterRESTRoutes performs a no-op as the EVM module doesn't expose REST
// endpoints
func (AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {
}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(c client.Context, serveMux *runtime.ServeMux) {
	if err := evm.RegisterQueryHandlerClient(context.Background(), serveMux, evm.NewQueryClient(c)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the evm module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns no root query command for the evm module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// RegisterInterfaces registers interfaces and implementations of the evm module.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	evm.RegisterInterfaces(registry)
	eth.RegisterInterfaces(registry)
}

// ____________________________________________________________________________

// AppModule implements an application module for the evm module.
type AppModule struct {
	AppModuleBasic
	keeper *evmkeeper.Keeper
	ak     evm.AccountKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(k *evmkeeper.Keeper, ak evm.AccountKeeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		ak:             ak,
	}
}

// Name returns the evm module's name.
func (AppModule) Name() string {
	return evm.ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// RegisterInvariants interface for registering invariants. Performs a no-op
// as the evm module doesn't expose invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	evm.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	evm.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// BeginBlock returns the begin block for the evm module.
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	am.keeper.BeginBlock(ctx, req)
}

// EndBlock returns the end blocker for the evm module. It returns no validator
// updates.
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return am.keeper.EndBlock(ctx, req)
}

// InitGenesis performs genesis initialization for the evm module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState evm.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	am.keeper.InitGenesis(ctx, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the evm
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// RegisterStoreDecoder registers a decoder for evm module's types
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {
}

// GenerateGenesisState creates a randomized GenState of the evm module.
func (AppModule) GenerateGenesisState(_ *module.SimulationState) {
}

// WeightedOperations returns the all the evm module operations with their respective weights.
func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

//
// App Wiring Setup
//

func init() {
	appmodule.Register(&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
	appmodule.Register(&bankmodulev1.Module{},
		appmodule.Provide(ProvideNibiruBankModule),
	)
}

type NibiruBankInputs struct {
	depinject.In

	Config *bankmodulev1.Module
	Cdc    codec.Codec
	Key    *store.KVStoreKey

	AccountKeeper banktypes.AccountKeeper

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace exported.Subspace `optional:"true"`
}

type NibiruBankOutputs struct {
	depinject.Out

	BankKeeper *evmkeeper.NibiruBankKeeper
	Module     appmodule.AppModule
}

func ProvideNibiruBankModule(in NibiruBankInputs) NibiruBankOutputs {
	// Configure blocked module accounts.
	//
	// Default behavior for blockedAddresses is to regard any module mentioned in
	// AccountKeeper's module account permissions as blocked.
	blockedAddresses := make(map[string]bool)
	if len(in.Config.BlockedModuleAccountsOverride) > 0 {
		for _, moduleName := range in.Config.BlockedModuleAccountsOverride {
			blockedAddresses[authtypes.NewModuleAddress(moduleName).String()] = true
		}
	} else {
		for _, permission := range in.AccountKeeper.GetModulePermissions() {
			blockedAddresses[permission.GetAddress().String()] = true
		}
	}

	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	nibiruBankKeeper := &evmkeeper.NibiruBankKeeper{
		BaseKeeper: bankkeeper.NewBaseKeeper(
			in.Cdc,
			in.Key,
			in.AccountKeeper,
			blockedAddresses,
			authority.String(),
		),
		StateDB: nil,
	}
	m := bank.NewAppModule(in.Cdc, nibiruBankKeeper, in.AccountKeeper, in.LegacySubspace)

	return NibiruBankOutputs{BankKeeper: nibiruBankKeeper, Module: m}
}

type EvmInputs struct {
	depinject.In

	Config       *modulev1.Module
	Key          *store.KVStoreKey
	TransientKey *store.TransientStoreKey
	Cdc          codec.Codec
	AppOpts      servertypes.AppOptions `optional:"true"`

	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	StakingKeeper evm.StakingKeeper
}

type EvmOutputs struct {
	depinject.Out

	Keeper *evmkeeper.Keeper
	Module appmodule.AppModule
}

func ProvideModule(in EvmInputs) EvmOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	k := evmkeeper.NewKeeper(in.Cdc, in.Key, in.TransientKey, authority, in.AccountKeeper, in.BankKeeper.(*evmkeeper.NibiruBankKeeper), in.StakingKeeper, cast.ToString(in.AppOpts.Get("evm.tracer")))

	m := NewAppModule(&k, in.AccountKeeper)

	return EvmOutputs{
		Keeper: &k,
		Module: m,
	}
}
