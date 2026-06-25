package upgrades

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
)

var _ HandlerImpl = (*Handler_v2_15)(nil)

type Handler_v2_15 struct{}

func (h Handler_v2_15) Handler(
	mm *module.Manager,
	cfg module.Configurator,
	nibiru *keepers.PublicKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		RunUpgrade2_15_0(nibiru, ctx)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

type Upgrade2_15_AddrCfg struct {
	ImplAdapterAddr sdk.AccAddress
}

var AddrCfg_v2_15 = Upgrade2_15_AddrCfg{
	// TODO: Replace with the deterministic instantiate2 address for the
	// x-oracle adapter contract once the final optimized Wasm bytecode,
	// instantiate message, creator, salt, and admin are fixed.
	ImplAdapterAddr: nil,
}

func RunUpgrade2_15_0(nibiru *keepers.PublicKeepers, ctx sdk.Context) {
	addrCfg := AddrCfg_v2_15
	if addrCfg.ImplAdapterAddr.Empty() {
		ctx.Logger().Info("oracle impl adapter address not configured in upgrade handler")
		return
	}

	nibiru.OracleKeeper.ImplAdapterAddr.Set(ctx, addrCfg.ImplAdapterAddr)
}
