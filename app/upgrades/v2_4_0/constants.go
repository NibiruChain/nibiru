package v2_4_0

import (
	"github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v2.4.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}
