package upgrades

import (
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func emptyUpgradeHandler(mm *module.Manager, configurator module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

type Upgrade struct {
	UpgradeName          string
	CreateUpgradeHandler func(*module.Manager, module.Configurator) upgradetypes.UpgradeHandler
	StoreUpgrades        types.StoreUpgrades
}

var Upgrade_0_21_10 = Upgrade{
	UpgradeName:          "0.21.10",
	CreateUpgradeHandler: emptyUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}

var Upgrade_0_21_11_alpha_1 = Upgrade{
	UpgradeName:          "0.21.11-alpha.1",
	CreateUpgradeHandler: emptyUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}
