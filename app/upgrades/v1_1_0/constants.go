package v1_1_0

import (
	"cosmossdk.io/store/types"
	"github.com/NibiruChain/nibiru/app/upgrades"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
)

const UpgradeName = "v1.1.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,

	CreateUpgradeHandler: upgrades.CreateUpgradeHandler,
	StoreUpgrades: types.StoreUpgrades{
		Added: []string{inflationtypes.ModuleName},
	},
}
