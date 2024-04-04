package upgrades

import (
	"context"
	store "cosmossdk.io/store/types"
	"cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func(module.Manager, module.Configurator) types.UpgradeHandler

	StoreUpgrades store.StoreUpgrades
}

func CreateUpgradeHandler(mm module.Manager, configurator module.Configurator) types.UpgradeHandler {
	return func(ctx context.Context, plan types.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
