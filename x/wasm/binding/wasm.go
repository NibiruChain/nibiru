package binding

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"

	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper/v1"
	"github.com/NibiruChain/nibiru/x/sudo"
)

func RegisterWasmOptions(
	perp perpkeeper.Keeper,
	perpAmm perpammkeeper.Keeper,
	sudoKeeper sudo.Keeper,
	oracleKeeper oraclekeeper.Keeper,
) []wasm.Option {
	wasmQueryPlugin := NewQueryPlugin(perp, perpAmm)
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})

	wasmExecuteOption := wasmkeeper.WithMessageHandlerDecorator(
		CustomExecuteMsgHandler(perp, sudoKeeper, oracleKeeper),
	)

	return []wasm.Option{wasmQueryOption, wasmExecuteOption}
}
