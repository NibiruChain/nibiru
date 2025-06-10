package v2_3_0

import (
	"cosmossdk.io/store/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v2.3.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}
