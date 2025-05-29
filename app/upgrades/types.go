package upgrades

import (
	"context"

	store "cosmossdk.io/store/types"
	"cosmossdk.io/x/upgrade/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/cosmos/cosmos-sdk/types/module"
	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func(
		mm *module.Manager,
		cfg module.Configurator,
		nibiru *keepers.PublicKeepers,
		ibcKeeperClientKeeper clientkeeper.Keeper,
	) types.UpgradeHandler

	StoreUpgrades store.StoreUpgrades
}

// DefaultUpgradeHandler runs module manager migrations without running any other
// logic that uses the Nibiru keepers. This is the most common value for
// the "CreateUpgradeHandler" field of an [Upgrade].
func DefaultUpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	nibiru *keepers.PublicKeepers,
	clientKeeper clientkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
