package v1_5_0

import (
	"cosmossdk.io/store/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v1.5.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}
