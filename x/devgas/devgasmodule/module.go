package devgasmodule

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	sdkcodectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/x/devgas"
	"github.com/NibiruChain/nibiru/v2/x/devgas/client/cli"
	"github.com/NibiruChain/nibiru/v2/x/devgas/exported"
	"github.com/NibiruChain/nibiru/v2/x/devgas/keeper"
	"github.com/NibiruChain/nibiru/v2/x/devgas/simulation"
)

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

// ConsensusVersion defines the current module consensus version.
const ConsensusVersion = 2

// AppModuleBasic type for the fees module
type AppModuleBasic struct{}

// Name returns the fees module's name.
func (AppModuleBasic) Name() string {
	return devgas.ModuleName
}

// RegisterLegacyAminoCodec performs a no-op as the fees do not support Amino
// encoding.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *sdkcodec.LegacyAmino) {
	devgas.RegisterLegacyAminoCodec(cdc)
}

// ConsensusVersion returns the consensus state-breaking version for the module.
func (AppModuleBasic) ConsensusVersion() uint64 {
	return ConsensusVersion
}

// RegisterInterfaces registers interfaces and implementations of the fees
// module.
func (AppModuleBasic) RegisterInterfaces(
	interfaceRegistry sdkcodectypes.InterfaceRegistry,
) {
	devgas.RegisterInterfaces(interfaceRegistry)
}

// DefaultGenesis returns default genesis state as raw bytes for the fees
// module.
func (AppModuleBasic) DefaultGenesis(cdc sdkcodec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(devgas.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the fees module.
func (b AppModuleBasic) ValidateGenesis(
	cdc sdkcodec.JSONCodec, _ sdkclient.TxEncodingConfig, bz json.RawMessage,
) error {
	var genesisState devgas.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genesisState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", devgas.ModuleName, err)
	}

	return genesisState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the fees
// module.
func (b AppModuleBasic) RegisterGRPCGatewayRoutes(
	c sdkclient.Context, serveMux *runtime.ServeMux,
) {
	if err := devgas.RegisterQueryHandlerClient(context.Background(), serveMux, devgas.NewQueryClient(c)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the fees module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd returns the fees module's root query command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// ___________________________________________________________________________

// AppModule implements the AppModule interface for the fees module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
	ak     authkeeper.AccountKeeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule Object
func NewAppModule(
	k keeper.Keeper,
	ak authkeeper.AccountKeeper,
	ss exported.Subspace,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		ak:             ak,
		legacySubspace: ss,
	}
}

// Name returns the fees module's name.
func (AppModule) Name() string {
	return devgas.ModuleName
}

// RegisterInvariants registers the fees module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// QuerierRoute returns the module's query routing key.
func (am AppModule) QuerierRoute() string {
	return devgas.RouterKey
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	devgas.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	devgas.RegisterQueryServer(
		cfg.QueryServer(), keeper.NewQuerier(am.keeper),
	)
}

// BeginBlock executes all ABCI BeginBlock logic respective to the fees module.
func (am AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {
}

// EndBlock executes all ABCI EndBlock logic respective to the fee-share module. It
// returns no validator updates.
func (am AppModule) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// InitGenesis performs the fees module's genesis initialization. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc sdkcodec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState devgas.GenesisState

	cdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.keeper, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the fees module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc sdkcodec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

//----------------------------------------------------------------------------
// AppModuleSimulation functions
//----------------------------------------------------------------------------

// GenerateGenesisState implements module.AppModuleSimulation.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// RegisterStoreDecoder implements module.AppModuleSimulation.
func (AppModule) RegisterStoreDecoder(sdk.StoreDecoderRegistry) {
}

// WeightedOperations implements module.AppModuleSimulation.
func (AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return nil
}
