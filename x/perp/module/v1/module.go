package perp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/x/perp"
	"github.com/NibiruChain/nibiru/x/perp/client/cli"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
)

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
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

func (AppModuleBasic) RegisterCodec(aminoCodec *codec.LegacyAmino) {
	types.RegisterCodec(aminoCodec)
}

func (AppModuleBasic) RegisterLegacyAminoCodec(aminoCodec *codec.LegacyAmino) {
	types.RegisterCodec(aminoCodec)
}

// DefaultGenesis returns default genesis state as raw bytes for the erc20
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the capability module.
func (AppModuleBasic) ValidateGenesis(
	cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage,
) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterRESTRoutes registers the capability module's REST service handlers.
func (AppModuleBasic) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
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

	keeper keeper.Keeper
	ak     types.AccountKeeper
	bk     types.BankKeeper
	ok     types.OracleKeeper
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ok types.OracleKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
		ak:             ak,
		bk:             bk,
		ok:             ok,
	}
}

// Name returns the capability module's name.
func (am AppModule) Name() string {
	return am.AppModuleBasic.Name()
}

// Route returns the capability module's message routing key.
func (am AppModule) Route() sdk.Route {
	return sdk.NewRoute(types.RouterKey, NewHandler(am.keeper))
}

// QuerierRoute returns the capability module's query routing key.
func (AppModule) QuerierRoute() string { return types.QuerierRoute }

// LegacyQuerierHandler returns the capability module's Querier.
func (am AppModule) LegacyQuerierHandler(legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return nil
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	err := cfg.RegisterMigration(types.ModuleName, 1, func(ctx sdk.Context) error { return nil }) // From 1 to 2
	if err != nil {
		panic(fmt.Errorf("failed to register migration: %w", err))
	}
	err = cfg.RegisterMigration(types.ModuleName, 2, keeper.From2To3(am.keeper, am.keeper.PerpAmmKeeper)) // From 2 to 3
	if err != nil {
		panic(fmt.Errorf("failed to register migration: %w", err))
	}
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

	perp.InitGenesis(ctx, am.keeper, genState)

	// See https://github.com/cosmos/cosmos-sdk/issues/5569 on why we do this.
	am.ak.GetModuleAccount(ctx, types.PerpEFModuleAccount)
	am.ak.GetModuleAccount(ctx, types.VaultModuleAccount)
	am.ak.GetModuleAccount(ctx, types.FeePoolModuleAccount)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the capability module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := perp.ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion implements ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 3 }

// BeginBlock executes all ABCI BeginBlock logic respective to the capability module.
func (am AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock executes all ABCI EndBlock logic respective to the capability module. It
// returns no validator updates.
func (am AppModule) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	EndBlocker(ctx, am.keeper)
	return []abci.ValidatorUpdate{}
}
