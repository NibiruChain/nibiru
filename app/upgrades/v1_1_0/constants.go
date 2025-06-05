package v1_1_0

import (
	"github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

const UpgradeName = "v1.1.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades: types.StoreUpgrades{
		Added: []string{inflationtypes.ModuleName},
	},
}
