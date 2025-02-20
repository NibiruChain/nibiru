package v1_4_0

import (
	"context"

	"cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/app/upgrades"
)

const UpgradeName = "v1.4.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: func(mm *module.Manager, cfg module.Configurator) upgradetypes.UpgradeHandler {
		return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			ctx := sdk.UnwrapSDKContext(c)
			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: types.StoreUpgrades{
		Added: []string{},
	},
}
