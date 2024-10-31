package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/upgrade/types"

	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func(*module.Manager, module.Configurator, clientkeeper.Keeper) types.UpgradeHandler

	StoreUpgrades store.StoreUpgrades
}
