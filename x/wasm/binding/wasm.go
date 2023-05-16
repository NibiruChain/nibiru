package binding

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper/v1"
	perpv2keeper "github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	"github.com/NibiruChain/nibiru/x/sudo"
)

func RegisterWasmOptions(
	perp *perpkeeper.Keeper,
	perpAmm *perpammkeeper.Keeper,
	perpv2 *perpv2keeper.Keeper,
	sudoKeeper *sudo.Keeper,
) []wasm.Option {
	// Custom querier
	wasmQueryPlugin := NewQueryPlugin(perp, perpAmm)
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})

	wasmExecuteOption := wasmkeeper.WithMessageHandlerDecorator(
		CustomExecuteMsgHandler(*perp, *perpv2, *sudoKeeper),
	)

	return []wasm.Option{wasmQueryOption, wasmExecuteOption}
}
