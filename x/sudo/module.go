package sudo

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/sudo/cli"
	sudokeeper "github.com/NibiruChain/nibiru/x/sudo/keeper"
	simulation "github.com/NibiruChain/nibiru/x/sudo/simulation"
	"github.com/NibiruChain/nibiru/x/sudo/types"
)

// Ensure the interface is properly implemented at compile time
var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

type AppModuleBasic struct {
	binaryCodec codec.BinaryCodec
}

func NewAppModuleBasic(binaryCodec codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{binaryCodec: binaryCodec}
}

func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterInterfaces registers interfaces and implementations of the perp module.
func (AppModuleBasic) RegisterInterfaces(interfaceRegistry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(interfaceRegistry)
}

func (AppModuleBasic) RegisterLegacyAminoCodec(aminoCodec *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(aminoCodec)
}

// DefaultGenesis returns default genesis state as raw bytes for the module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the capability module.
func (AppModuleBasic) ValidateGenesis(
	cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage,
) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(
	clientCtx client.Context, mux *runtime.ServeMux,
) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the capability module's root tx command.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the capability module's root query command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface for the module.
type AppModule struct {
	AppModuleBasic

	keeper sudokeeper.Keeper
}

func NewAppModule(
	cdc codec.Codec,
	keeper sudokeeper.Keeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
	}
}

// Name returns the capability module's name.
func (am AppModule) Name() string {
	return am.AppModuleBasic.Name()
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), sudokeeper.NewQuerier(am.keeper))
	types.RegisterMsgServer(cfg.MsgServer(), sudokeeper.NewMsgServer(am.keeper))
}

// RegisterInvariants registers the capability module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the capability module's genesis initialization It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage,
) []abci.ValidatorUpdate {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the capability module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion implements ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 3 }

// BeginBlock executes all ABCI BeginBlock logic respective to the capability module.
func (am AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock executes all ABCI EndBlock logic respective to the capability module. It
// returns no validator updates.
func (am AppModule) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
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
	appParams := simState.AppParams

	fmt.Println("Map:", fmt.Sprintf("%v", appParams))
	for k, v := range appParams {
		var val int
		fmt.Printf("Key: %s, Value: %d\n", k, v)
		err := json.Unmarshal(v, &val)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Key: %s, Value: %d\n", k, val)
	}

	return nil
}
