// Package consensus is Nibiru's x/consensus module. It owns the store that
// holds Nibiru BFT's consensus params, embeds the x-consensus CosmWasm
// contract, creates it, and copies its params into that store once per block.
//
// Layout: types holds the messages, store keys, module constants and the
// contract's Go types; keeper holds the Keeper; this package holds the
// AppModule and the embedded contract build (ContractWasm).
//
// This module replaces the Cosmos SDK x/consensus module in the app with the
// same name, store key, value key, messages and query, so a live chain needs
// no store migration. The SDK module in lib/cosmos-sdk/x/consensus is
// deprecated and only serves lib's own test apps.
//
// x-consensus owns the consensus params, the role the Cosmos SDK module used
// to have. It holds the block, evidence and validator params plus
// min_block_commit_duration_ms, a chain-wide commit wait that replaces each
// node's config.toml timeout_commit. The x/consensus store remains only as
// the copy that baseapp and Nibiru BFT read, and MsgUpdateParams fails with
// types.ErrUpdateParamsDisabled.
//
// Only the current x/sudo root may change the contract. It reads the root on
// every execute through the Stargate query /nibiru.sudo.v1.Query/QuerySudoers
// and stores no owner, so a root rotation applies at once. The contract is
// also its own Wasm admin, so the root migrates it through the contract
// (types.ExecuteMsg.Migrate) and migration rights follow the live root too.
// Governance stays the break-glass for migration.
//
// The module EndBlocker (AppModule, ordered last) queries the contract once
// per block under types.QueryGasLimit, validates the result with Nibiru BFT's
// own rules, and writes it to the x/consensus store. baseapp reads that store
// at BeginBlock, for every tx, and at EndBlock, where it returns the params
// to Nibiru BFT. So a change executed in block H applies from H+1 for both
// the app and the engine, and a failed query keeps the last good copy. The
// app version is not in the contract: x/upgrade owns it, and the refresh
// copies it unchanged. Contract source: wasm-contracts/x-consensus.
package consensus

import (
	"context"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/v2/x/consensus/types"
)

var (
	_ module.AppModule         = AppModule{}
	_ module.HasServices       = AppModule{}
	_ module.EndBlockAppModule = AppModule{}
)

// Keeper is the part of keeper.Keeper the module uses. It is an interface
// because the keeper package imports this one for ContractWasm, so this
// package cannot import the keeper back.
type Keeper interface {
	// Refresh copies the contract params into the x/consensus store.
	Refresh(ctx sdk.Context) error
	// QueryServer serves the Params query from the x/consensus store.
	types.QueryServer
}

// AppModule is the x/consensus module. It serves the Params query, rejects
// MsgUpdateParams, and runs Keeper.Refresh at the end of every block. The
// app must order its EndBlocker last, after every module that can execute
// the contract in the block (gov proposals run in the gov EndBlocker), so
// the copy reflects the whole block before baseapp returns it to Nibiru BFT.
//
// It has no genesis: baseapp stores the consensus params from InitChain and
// exports them from the store, as it did with the Cosmos SDK module, and the
// app creates the contract after module genesis.
//
// It is not wired through depinject and does not call appmodule.Register, so
// it never collides with the deprecated lib/cosmos-sdk/x/consensus depinject
// module when both are linked into one test binary. The app mounts the store
// and sets the Keeper as baseapp's param store itself.
//
// The EndBlocker is a module, not code in NibiruApp.EndBlocker, because the
// runtime package installs its own EndBlocker over the app's.
type AppModule struct {
	keeper Keeper
}

// NewAppModule returns the module for keeper.
func NewAppModule(keeper Keeper) AppModule { return AppModule{keeper: keeper} }

// Name returns types.ModuleName, the name the Cosmos SDK module used.
func (AppModule) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers MsgUpdateParams for amino JSON, so old
// txs and gov proposals that carry it still decode.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers MsgUpdateParams and the Msg service, so old
// txs and gov proposals that carry it still decode and reach the disabled
// handler.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// RegisterGRPCGatewayRoutes registers the REST route of the Params query.
func (AppModule) RegisterGRPCGatewayRoutes(
	clientCtx client.Context, mux *gwruntime.ServeMux,
) {
	if err := types.RegisterQueryHandlerClient(
		context.Background(), mux, types.NewQueryClient(clientCtx),
	); err != nil {
		panic(err)
	}
}

// RegisterServices registers the Params query and the disabled Msg service.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), msgServer{})
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// GetTxCmd returns nil: updates are contract executions.
func (AppModule) GetTxCmd() *cobra.Command { return nil }

// GetQueryCmd returns nil, like the Cosmos SDK module: use the gRPC or REST
// Params query, or query the contract.
func (AppModule) GetQueryCmd() *cobra.Command { return nil }

// ConsensusVersion is 1, the version of the Cosmos SDK module. The store
// layout is unchanged, so there is nothing to migrate.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// EndBlock refreshes the x/consensus store from x-consensus. A failure is
// logged and the stored params stay as they were; it never panics, so a
// broken contract cannot halt the chain. Refresh writes at most one store
// key, after every check passed, so a recovered panic cannot leave a partial
// write.
func (am AppModule) EndBlock(
	ctx sdk.Context, _ abci.RequestEndBlock,
) []abci.ValidatorUpdate {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error(
				"x-consensus refresh panicked; keeping stored consensus params",
				"module", types.ModuleName, "height", ctx.BlockHeight(),
				"panic", r)
		}
	}()
	if err := am.keeper.Refresh(ctx); err != nil {
		ctx.Logger().Error(
			"x-consensus refresh failed; keeping stored consensus params",
			"module", types.ModuleName, "height", ctx.BlockHeight(), "err", err)
	}
	return nil
}

// msgServer rejects MsgUpdateParams with types.ErrUpdateParamsDisabled.
//
// The rejection lives in the module's own Msg server rather than a router
// circuit breaker, so it needs no app-wide hook and covers every path through
// the router: txs, gov proposals (executed in the gov EndBlocker, where an
// ante filter would not see them), authz, Wasm and EVM dispatch. The Refresh
// in the same block would also overwrite any write, so this is defense in
// depth.
type msgServer struct{}

var _ types.MsgServer = msgServer{}

// UpdateParams always fails. See msgServer.
func (msgServer) UpdateParams(
	context.Context, *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	return nil, types.ErrUpdateParamsDisabled
}
