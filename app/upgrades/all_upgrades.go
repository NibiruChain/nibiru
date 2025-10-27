package upgrades

import (
	"github.com/NibiruChain/nibiru/v2/app/keepers"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
)

type Upgrade struct {
	UpgradeName string

	CreateUpgradeHandler func(
		mm *module.Manager,
		cfg module.Configurator,
		nibiru *keepers.PublicKeepers,
		ibcKeeperClientKeeper clientkeeper.Keeper,
	) upgradetypes.UpgradeHandler

	// StoreUpgrades defines a series of transformations to apply the multistore db
	// upon load
	StoreUpgrades store.StoreUpgrades
}

var AllUpgrades = []Upgrade{
	Upgrade1_0_1,
	Upgrade1_0_2,
	Upgrade1_0_3,
	Upgrade1_1_0,
	Upgrade1_2_0,
	Upgrade1_3_0,
	Upgrade1_4_0,
	Upgrade1_5_0,
	Upgrade2_0_0,
	Upgrade2_1_0,
	Upgrade2_2_0,
	Upgrade2_3_0,
	Upgrade2_4_0,
	Upgrade2_5_0,
	Upgrade2_6_0,
	Upgrade2_7_0,
	Upgrade2_8_0,
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
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
