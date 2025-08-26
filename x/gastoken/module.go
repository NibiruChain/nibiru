/*
The gastoken modules allows nodes to easily support many
tokens for usage, while letting node operators
only specify their tx fee parameters for a single "base" asset.
*/
package gastoken

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/keeper"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	modulev1 "github.com/NibiruChain/nibiru/v2/api/nibiru/gastoken/module"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/gastoken/cli"
)

var (
	_ module.AppModuleBasic   = AppModuleBasic{}
	_ module.HasGenesisBasics = AppModuleBasic{}

	_ module.AppModule           = AppModule{}
	_ module.HasConsensusVersion = AppModule{}
	_ module.HasServices         = AppModule{}
)

const ModuleName = types.ModuleName

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface for the gastoken module.
type AppModuleBasic struct{}

func NewAppModuleBasic() AppModuleBasic {
	return AppModuleBasic{}
}

// Name returns the gastoken module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the gastoken module's default genesis state.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the txfee module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	//nolint:errcheck
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}

// GetTxCmd returns the gastoken module's root tx command.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the gastoken module's root query command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface for the gastoken module.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
}

func NewAppModule(keeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(),
		keeper:         keeper,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType is a marker function just indicates that this is a one-per-module type.
func (am AppModule) IsOnePerModuleType() {}

// Name returns the gastoken module's name.
func (am AppModule) Name() string {
	return am.AppModuleBasic.Name()
}

// QuerierRoute returns the gastoken module's query routing key.
func (AppModule) QuerierRoute() string { return types.RouterKey }

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))
}

// RegisterInvariants registers the gastoken module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the gastoken module's genesis initialization It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	am.keeper.InitGenesis(ctx, genState)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the gastoken module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(genState)
}

// BeginBlock executes all ABCI BeginBlock logic respective to the gastoken module.
func (AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

func (am AppModule) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

//
// App Wiring Setup
//

func init() {
	appmodule.Register(&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type GasTokenInputs struct {
	depinject.In

	Config *modulev1.Module
	Key    *store.KVStoreKey
	Cdc    codec.Codec

	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    types.BankKeeper
	EvmKeeper     *evmkeeper.Keeper
	SudoKeeper    types.SudoKeeper
}

type GasTokenOutputs struct {
	depinject.Out

	Keeper keeper.Keeper

	Module appmodule.AppModule
}

func ProvideModule(in GasTokenInputs) GasTokenOutputs {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}
	k := keeper.NewKeeper(in.Cdc, in.Key, in.AccountKeeper, in.BankKeeper, in.EvmKeeper, in.SudoKeeper, authority.String())

	m := NewAppModule(k)

	return GasTokenOutputs{
		Keeper: k,
		Module: m,
	}
}
