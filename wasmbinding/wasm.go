package wasmbinding

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/x/sudo/keeper"

	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
)

// NibiruWasmOptions: Wasm Options are extension points to instantiate the Wasm
// keeper with non-default values
func NibiruWasmOptions(
	grpcQueryRouter *baseapp.GRPCQueryRouter,
	appCodec codec.Codec,
	sudoKeeper keeper.Keeper,
	oracleKeeper oraclekeeper.Keeper,
) []wasm.Option {
	wasmQueryPlugin := NewQueryPlugin(oracleKeeper)
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
		Stargate: wasmkeeper.AcceptListStargateQuerier(
			WasmAcceptedStargateQueries(),
			grpcQueryRouter,
			appCodec,
		),
	})

	wasmExecuteOption := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(sudoKeeper, oracleKeeper),
	)

	return []wasm.Option{wasmQueryOption, wasmExecuteOption}
}
