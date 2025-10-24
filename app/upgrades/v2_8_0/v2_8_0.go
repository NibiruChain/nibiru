package v2_8_0

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v2.8.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}
