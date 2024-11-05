package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func(*module.Manager, module.Configurator) types.UpgradeHandler

	StoreUpgrades store.StoreUpgrades
}
