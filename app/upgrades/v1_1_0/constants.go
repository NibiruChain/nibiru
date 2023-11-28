package v1_1_0

import (
	"github.com/NibiruChain/nibiru/app/upgrades"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
	"github.com/cosmos/cosmos-sdk/store/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v1.1.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: func() upgradetypes.UpgradeHandler {
		return nil
	},
	StoreUpgrades: types.StoreUpgrades{
		Added: []string{inflationtypes.ModuleName},
	},
}
