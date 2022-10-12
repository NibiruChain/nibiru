package util

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/x/util/client/cli"
	"github.com/NibiruChain/nibiru/x/util/keeper"
	utiltypes "github.com/NibiruChain/nibiru/x/util/types"
)

var (
	_ module.AppModule = AppModule{}
)

type AppModule struct {
	bankKeeper utiltypes.BankKeeper
}

func NewAppModule(bk utiltypes.BankKeeper) *AppModule {
	return &AppModule{
		bankKeeper: bk,
	}
}

func (a AppModule) Name() string {
	return "util"
}

func (a AppModule) RegisterLegacyAminoCodec(*codec.LegacyAmino)     {}
func (a AppModule) RegisterInterfaces(codectypes.InterfaceRegistry) {}
func (a AppModule) DefaultGenesis(codec.JSONCodec) json.RawMessage  { return nil }
func (a AppModule) ValidateGenesis(codec.JSONCodec, client.TxEncodingConfig, json.RawMessage) error {
	return nil
}
func (a AppModule) RegisterRESTRoutes(client.Context, *mux.Router)              {}
func (a AppModule) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {}
func (a AppModule) GetTxCmd() *cobra.Command                                    { return nil }
func (a AppModule) GetQueryCmd() *cobra.Command                                 { return cli.GetQueryCmd() }
func (a AppModule) InitGenesis(sdk.Context, codec.JSONCodec, json.RawMessage) []abci.ValidatorUpdate {
	return nil
}
func (a AppModule) ExportGenesis(sdk.Context, codec.JSONCodec) json.RawMessage {
	return nil
}
func (a AppModule) RegisterInvariants(sdk.InvariantRegistry) {}
func (a AppModule) Route() sdk.Route {
	return sdk.NewRoute("", NewHandler())
}
func (a AppModule) QuerierRoute() string {
	return ""
}
func (a AppModule) LegacyQuerierHandler(*codec.LegacyAmino) sdk.Querier {
	return nil
}
func (a AppModule) RegisterServices(cfg module.Configurator) {
	utiltypes.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(a.bankKeeper))
}
func (a AppModule) ConsensusVersion() uint64 {
	return 1
}
func (a AppModule) BeginBlock(sdk.Context, abci.RequestBeginBlock) {}
func (a AppModule) EndBlock(sdk.Context, abci.RequestEndBlock) []abci.ValidatorUpdate {
	return nil
}
