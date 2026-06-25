package upgrades

import (
	"github.com/NibiruChain/nibiru/v2/app/keepers"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// Upgrade describes the behavior for a named Nibiru software upgrade and the
// upgrade plan (See [upgradetypes.Plan]).
type Upgrade struct {
	// UpgradeName must match the name in the on-chain software upgrade plan.
	// The upgrade keeper uses this value to register the handler and the store
	// loader for the height specified by governance.
	UpgradeName string

	// Handler builds the upgrade handler that runs at the upgrade height. Use a
	// custom handler when the upgrade needs app-specific state changes before
	// module migrations. Otherwise use [DefaultUpgradeHandler].
	Handler HandlerImpl

	// StoreUpgrades declares store keys added, renamed, or deleted when the node
	// loads the upgraded binary. Leave this empty when the upgrade does not
	// change the multistore layout.
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
	Upgrade2_13_0,
	Upgrade2_14_0,
	Upgrade2_14_1,
}

// HandlerImpl is a struct wrapper for custom upgrade handler implementations.
type HandlerImpl interface {
	Handler(
		mm *module.Manager,
		cfg module.Configurator,
		nibiru *keepers.PublicKeepers,
	) upgradetypes.UpgradeHandler
}

// NewVanillaUpgrade returns an [Upgrade] that only performs standard module
// consensus version migrations without running any custom logic that uses the
// Nibiru keepers. Standard releases that bump the binary to a new version and
// modify the state machine are considered "vanilla" upgrades.
func NewVanillaUpgrade(upgradeName string) Upgrade {
	return Upgrade{
		UpgradeName:   upgradeName,
		Handler:       DefaultUpgradeHandler{},
		StoreUpgrades: store.StoreUpgrades{},
	}
}

// DefaultUpgradeHandler runs module manager migrations without running any
// other logic that uses the Nibiru keepers.
type DefaultUpgradeHandler struct{}

var _ HandlerImpl = (*DefaultUpgradeHandler)(nil)

// Handler for [DefaultUpgradeHandler] runs module manager migrations without
// running any other logic that uses the Nibiru keepers.
func (h DefaultUpgradeHandler) Handler(
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
