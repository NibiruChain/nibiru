// Package consensus is Nibiru's x/consensus module. It embeds the x-consensus
// CosmWasm contract, creates it, and copies its params into the x/consensus
// store once per block.
//
// Layout: types holds the messages, store keys, module constants and the
// contract's Go types; keeper holds the Keeper; this package holds the
// AppModule and the embedded contract build (ContractWasm).
//
// x-consensus owns Nibiru BFT's consensus params, the role the x/consensus
// module used to have. It holds the block, evidence and validator params
// plus min_block_commit_duration_ms, a chain-wide commit wait that replaces
// each node's config.toml timeout_commit. The x/consensus store remains only
// as the copy that baseapp and Nibiru BFT read, and its MsgUpdateParams is
// blocked by a circuit breaker (app/xconsensus.go).
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
// own rules, and writes it to the x/consensus store. baseapp reads that store at
// BeginBlock, for every tx, and at EndBlock, where it returns the params to
// Nibiru BFT. So a change executed in block H applies from H+1 for both the
// app and the engine, and a failed query keeps the last good copy. The app
// version is not in the contract: x/upgrade owns it, and the refresh copies
// it unchanged. Contract source: wasm-contracts/x-consensus.
package consensus

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
)

var (
	_ module.AppModule         = AppModule{}
	_ module.EndBlockAppModule = AppModule{}
)

// AppModuleName is the module manager name of AppModule while the Cosmos SDK
// x/consensus module still occupies types.ModuleName in the app. The module
// manager rejects a second module with the same name.
const AppModuleName = "xconsensus"

// Keeper is the part of keeper.Keeper the module uses. It is an interface
// because the keeper package imports this one for ContractWasm, so this
// package cannot import the keeper back.
type Keeper interface {
	// Refresh copies the contract params into the x/consensus store.
	Refresh(ctx sdk.Context) error
}

// AppModule runs Keeper.Refresh at the end of every block. It has no store,
// genesis, messages or queries: the contract owns the params and the
// x/consensus store holds the copy. The app must order its EndBlocker last,
// after every module that can execute the contract in the block (gov proposals
// run in the gov EndBlocker), so the copy reflects the whole block before
// baseapp returns it to Nibiru BFT.
//
// It is a module, not code in NibiruApp.EndBlocker, because the runtime
// package installs its own EndBlocker over the app's.
type AppModule struct {
	keeper Keeper
}

// NewAppModule returns the module for keeper.
func NewAppModule(keeper Keeper) AppModule { return AppModule{keeper: keeper} }

// Name returns AppModuleName.
func (AppModule) Name() string { return AppModuleName }

// RegisterLegacyAminoCodec is a no-op: the module has no messages.
func (AppModule) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

// RegisterInterfaces is a no-op: the module has no messages.
func (AppModule) RegisterInterfaces(codectypes.InterfaceRegistry) {}

// RegisterGRPCGatewayRoutes is a no-op: the module has no queries. Read the
// params through x/consensus or the contract.
func (AppModule) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {}

// GetTxCmd returns nil: updates are contract executions.
func (AppModule) GetTxCmd() *cobra.Command { return nil }

// GetQueryCmd returns nil: query the contract or x/consensus.
func (AppModule) GetQueryCmd() *cobra.Command { return nil }

// ConsensusVersion is 1. The module has no store to migrate; the version only
// exists so the upgrade version map records the module.
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
				"module", AppModuleName, "height", ctx.BlockHeight(), "panic", r)
		}
	}()
	if err := am.keeper.Refresh(ctx); err != nil {
		ctx.Logger().Error(
			"x-consensus refresh failed; keeping stored consensus params",
			"module", AppModuleName, "height", ctx.BlockHeight(), "err", err)
	}
	return nil
}
