package wasmbinding

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	"github.com/NibiruChain/nibiru/x/sudo/keeper"

	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
)

func RegisterWasmOptions(
	sudoKeeper keeper.Keeper,
	oracleKeeper oraclekeeper.Keeper,
) []wasm.Option {
	wasmQueryPlugin := NewQueryPlugin(oracleKeeper)
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})

	wasmExecuteOption := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(sudoKeeper, oracleKeeper),
	)

	return []wasm.Option{wasmQueryOption, wasmExecuteOption}
}
