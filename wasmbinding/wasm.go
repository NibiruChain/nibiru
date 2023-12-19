package wasmbinding

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/x/sudo/keeper"

	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	perpv2keeper "github.com/NibiruChain/nibiru/x/perp/v2/keeper"
)

// NibiruWasmOptions: Wasm Options are extension points to instantiate the Wasm
// keeper with non-default values
func NibiruWasmOptions(
	grpcQueryRouter *baseapp.GRPCQueryRouter,
	appCodec codec.Codec,
	perpv2 perpv2keeper.Keeper,
	sudoKeeper keeper.Keeper,
	oracleKeeper oraclekeeper.Keeper,
) []wasmkeeper.Option {
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Stargate: wasmkeeper.AcceptListStargateQuerier(
			WasmAcceptedStargateQueries(),
			grpcQueryRouter,
			appCodec,
		),
	})

	wasmExecuteOption := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(perpv2, sudoKeeper, oracleKeeper),
	)

	return []wasmkeeper.Option{wasmQueryOption, wasmExecuteOption}
}
