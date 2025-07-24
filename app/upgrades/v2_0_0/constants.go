package v2_0_0

import (
	"cosmossdk.io/store/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	evmtypes "github.com/NibiruChain/nibiru/v2/x/evm"
)

const UpgradeName = "v2.0.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades: types.StoreUpgrades{
		Added: []string{evmtypes.ModuleName},
	},
}
