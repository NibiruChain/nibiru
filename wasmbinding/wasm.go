package wasmbinding

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/x/sudo/keeper"
)

// NibiruWasmOptions: Wasm Options are extension points to instantiate the Wasm
// keeper with non-default values
func NibiruWasmOptions(
	grpcQueryRouter *baseapp.GRPCQueryRouter,
	appCodec codec.Codec,
	sudoKeeper keeper.Keeper,
) []wasmkeeper.Option {
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Stargate: wasmkeeper.AcceptListStargateQuerier(
			WasmAcceptedStargateQueries(),
			grpcQueryRouter,
			appCodec,
		),
	})

	return []wasmkeeper.Option{wasmQueryOption}
}
