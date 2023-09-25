package tokenfactory

/*
The tokenfactory module allows any account to create a new token with
the name `factory/{creator address}/{subdenom}`.

- Mint and burn user denom to and form any account
- Create a transfer of their denom between any two accounts
- Change the admin.
*/

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	sdkcodectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	// modulecli "github.com/NibiruChain/nibiru/x/tokenfactory/client/cli"
	modulekeeper "github.com/NibiruChain/nibiru/x/tokenfactory/keeper"
	moduletypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// type check to ensure the interface is properly implemented
var (
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModule           = AppModule{}
	_ module.BeginBlockAppModule = AppModule{}
	_ module.EndBlockAppModule   = AppModule{}
)

// ConsensusVersion defines the current module consensus version.
const ConsensusVersion = 2

// AppModuleBasic type for the fees module
type AppModuleBasic struct{}

// Name returns the fees module's name.
func (AppModuleBasic) Name() string {
	return moduletypes.ModuleName
}

// RegisterLegacyAminoCodec performs a no-op as the fees do not support Amino
// encoding.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *sdkcodec.LegacyAmino) {
	moduletypes.RegisterLegacyAminoCodec(cdc)
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
	moduletypes.RegisterInterfaces(interfaceRegistry)
}

// DefaultGenesis returns default genesis state as raw bytes for the fees
// module.
func (AppModuleBasic) DefaultGenesis(cdc sdkcodec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(moduletypes.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the fees module.
func (b AppModuleBasic) ValidateGenesis(
	cdc sdkcodec.JSONCodec, _ sdkclient.TxEncodingConfig, bz json.RawMessage,
) error {
	var genesisState moduletypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genesisState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", moduletypes.ModuleName, err)
	}

	return genesisState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the fees
// module.
func (b AppModuleBasic) RegisterGRPCGatewayRoutes(
	c sdkclient.Context, serveMux *runtime.ServeMux,
) {
	if err := moduletypes.RegisterQueryHandlerClient(context.Background(), serveMux, moduletypes.NewQueryClient(c)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the fees module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	panic("TODO")
	// return modulecli.NewTxCmd()
}

// GetQueryCmd returns the fees module's root query command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	panic("TODO")
	// return modulecli.GetQueryCmd()
}

// ___________________________________________________________________________

// AppModule implements the AppModule interface for the fees module.
type AppModule struct {
	AppModuleBasic
	keeper modulekeeper.Keeper
	ak     authkeeper.AccountKeeper
}

// NewAppModule creates a new AppModule Object
func NewAppModule(
	k modulekeeper.Keeper,
	ak authkeeper.AccountKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		ak:             ak,
	}
}

// Name returns the fees module's name.
func (AppModule) Name() string {
	return moduletypes.ModuleName
}

// RegisterInvariants registers the fees module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// QuerierRoute returns the module's query routing key.
func (am AppModule) QuerierRoute() string {
	return moduletypes.RouterKey
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	moduletypes.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	moduletypes.RegisterQueryServer(
		cfg.QueryServer(), am.keeper.Querier(),
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
	var genesisState moduletypes.GenesisState

	cdc.MustUnmarshalJSON(data, &genesisState)
	am.keeper.InitGenesis(ctx, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the fees module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc sdkcodec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}
