package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"cosmossdk.io/x/upgrade/types"
	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func(*module.Manager, module.Configurator, clientkeeper.Keeper) types.UpgradeHandler

	StoreUpgrades storetypes.StoreUpgrades
}
