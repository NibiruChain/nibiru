package app

import (
	"github.com/spf13/cast"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	servertypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/upgrade"
	upgradekeeper "github.com/NibiruChain/nibiru/v2/x/upgrade/keeper"
)

// initUpgradeKeeper constructs the upgrade keeper before IBC consumes it.
// InitChainer supplies the complete module version map after all modules are
// registered, so this keeper does not need the depinject-only init version map.
func (app *NibiruApp) initUpgradeKeeper(appOpts servertypes.AppOptions) {
	skipUpgradeHeights := make(map[int64]bool)
	var homePath string
	if appOpts != nil {
		for _, height := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
			skipUpgradeHeights[int64(height)] = true
		}
		homePath = cast.ToString(appOpts.Get(flags.FlagHome))
	}

	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		app.keys[upgradetypes.StoreKey],
		app.appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.UpgradeKeeper.BindWasm(&app.WasmKeeper, app.SudoKeeper, upgrade.ContractWasm)
}
