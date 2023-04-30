package binding

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
)

func RegisterWasmOptions(
	perp *perpkeeper.Keeper,
	perpAmm *perpammkeeper.Keeper,
) []wasm.Option {
	// Custom querier
	wasmQueryPlugin := NewQueryPlugin(perp, perpAmm)
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})

	wasmExecuteOption := wasmkeeper.WithMessageHandlerDecorator(
		CustomExecuteMsgHandler(*perp),
	)

	return []wasm.Option{wasmQueryOption, wasmExecuteOption}
}
