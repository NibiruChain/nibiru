package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func() types.UpgradeHandler

	StoreUpgrades store.StoreUpgrades
}
