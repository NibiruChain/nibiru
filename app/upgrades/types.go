package upgrades

import (
	store "cosmossdk.io/store/types"
	"cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func(*module.Manager, module.Configurator) types.UpgradeHandler

	StoreUpgrades store.StoreUpgrades
}
