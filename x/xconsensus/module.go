package xconsensus

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
	keeper *Keeper
}

// NewAppModule returns the module for keeper.
func NewAppModule(keeper *Keeper) AppModule { return AppModule{keeper: keeper} }

// Name returns ModuleName.
func (AppModule) Name() string { return ModuleName }

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
				"module", ModuleName, "height", ctx.BlockHeight(), "panic", r)
		}
	}()
	if err := am.keeper.Refresh(ctx); err != nil {
		ctx.Logger().Error(
			"x-consensus refresh failed; keeping stored consensus params",
			"module", ModuleName, "height", ctx.BlockHeight(), "err", err)
	}
	return nil
}
