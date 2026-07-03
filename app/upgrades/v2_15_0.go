package upgrades

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/evm"
)

var _ HandlerImpl = (*Handler_v2_15)(nil)

type Handler_v2_15 struct{}

func (h Handler_v2_15) Handler(
	mm *module.Manager,
	cfg module.Configurator,
	nibiru *keepers.PublicKeepers,
) upgradetypes.UpgradeHandler {
	return func(
		ctx sdk.Context,
		plan upgradetypes.Plan,
		fromVM module.VersionMap,
	) (module.VersionMap, error) {
		err := h.runUpgrade2_15_0(nibiru, ctx)
		if err != nil {
			ctx.Logger().Error("v2.15.0 upgrade failure", "err", err)
			ctx.EventManager().EmitEvent(
				NewEventUpgradeFailure("v2.15.0", err),
			)
		}
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

// XOracleAdapterAddr_v2_15 is the deterministic instantiate2 address of the
// Sai x-oracle Wasm adapter. The bytecode is release artifact
// x_oracle.wasm from sai-perps tag wasm-contracts/v1.22.0, source commit
// 8f8a2e9040c3894b5f607a661190e229ba401a88, with data hash
// 1494ebbb5623d4e4860e76cb592f17b79612c975e3331437027e52802d2358bf.
//
// The same address exists on mainnet and testnet because instantiate2 used
// the same creator (Foundation Treasury CW3), salt ("x-oracle-nibiru"),
// optimized Wasm code hash, and fixed init JSON. The contract config differs
// by chain only after testnet UpdateConfig points it at the testnet Sai oracle.
const (
	XOracleAdapterAddr_v2_15 = "nibi1jc6cvkzj73mz685kfdydhugnx4evhn6mupngvhpmw5sapr98mp2sdshh3r"
)

func (h Handler_v2_15) runUpgrade2_15_0(
	nibiru *keepers.PublicKeepers,
	ctx sdk.Context,
) error {
	params := nibiru.EvmKeeper.GetParams(ctx)
	wasmPlugins := make([]evm.WasmPlugin, 0, len(params.WasmPlugins)+1)
	for _, plugin := range params.WasmPlugins {
		if plugin.Name != evm.WasmPluginNameXOracle {
			wasmPlugins = append(wasmPlugins, plugin)
		}
	}
	wasmPlugins = append(wasmPlugins, evm.WasmPlugin{
		Name: evm.WasmPluginNameXOracle,
		Addr: XOracleAdapterAddr_v2_15,
	})
	params.WasmPlugins = wasmPlugins
	if err := nibiru.EvmKeeper.SetParams(ctx, params); err != nil {
		return fmt.Errorf("failed to configure x-oracle wasm plugin: %w", err)
	}
	return nil
}
