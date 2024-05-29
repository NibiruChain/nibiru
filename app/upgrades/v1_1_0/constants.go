package v1_1_0

import (
	"context"

	"cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/app/upgrades"
	// inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
)

const UpgradeName = "v1.1.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: func(mm *module.Manager, cfg module.Configurator) upgradetypes.UpgradeHandler {
		return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: types.StoreUpgrades{
		Added: []string{},
		// Added: []string{inflationtypes.ModuleName},
	},
}
