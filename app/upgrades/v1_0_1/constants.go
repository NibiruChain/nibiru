package v1_0_1

import (
	"github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v1.0.1"

// pretty much a no-op store upgrade to test the upgrade process and include the newer version of rocksdb
var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}
