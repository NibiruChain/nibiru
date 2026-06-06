package upgrades

import (
	"github.com/NibiruChain/nibiru/v2/app/keepers"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

type Upgrade struct {
	UpgradeName string

	Handler HandlerImpl

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
	Upgrade2_9_0,
	Upgrade2_10_0,
	Upgrade2_11_0,
	Upgrade2_12_0,
	Upgrade2_14_0,
}

// HandlerImpl is a struct wrapper
type HandlerImpl interface {
	Handler(
		mm *module.Manager,
		cfg module.Configurator,
		nibiru *keepers.PublicKeepers,
	) upgradetypes.UpgradeHandler
}

// DefaultUpgraderHandler runs module manager migrations without running any
// other logic that uses the Nibiru keepers.
type DefaultUpgraderHandler struct{}

var _ HandlerImpl = (*DefaultUpgraderHandler)(nil)

// Handler for [DefaultUpgraderHandler] runs module manager migrations without
// running any other logic that uses the Nibiru keepers.
func (h DefaultUpgraderHandler) Handler(
	mm *module.Manager,
	cfg module.Configurator,
	nibiru *keepers.PublicKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

// NewEventUpgradeFailure builds an [sdk.Event] to use when an upgrade fails and
// exits. This is used to debug why an upgrade failed while allowing it to
// proceed without panicking and halting the blockchain.
func NewEventUpgradeFailure(upgradeName string, err error) sdk.Event {
	return sdk.NewEvent(
		"upgrade_failure",
		sdk.NewAttribute("upgrade", upgradeName),
		sdk.NewAttribute("error", err.Error()),
	)
}
