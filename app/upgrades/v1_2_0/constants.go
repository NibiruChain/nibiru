package v1_2_0

import (
	"context"

	"cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v1.2.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: func(mm *module.Manager, cfg module.Configurator, clientKeeper clientkeeper.Keeper) upgradetypes.UpgradeHandler {
		return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			ctx := sdk.UnwrapSDKContext(c)
			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: types.StoreUpgrades{},
}
