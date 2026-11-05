package evmtest

import (
	"fmt"
	"time"

	codec "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

// RunUpgrade invokes the full handler with a valid plan and persisted module
// versions and the app configurator containing registered migrations. Like
// UpgradeKeeper.ApplyUpgrade, it records returned versions only
// after success so repeated calls do not rerun completed migrations.
func (deps *TestDeps) RunUpgrade(upgrade upgrades.Upgrade) error {
	var (
		// ---- Run the upgrade handler. ----
		upgradeHandler = upgrade.Handler.Handler(
			deps.App.ModuleManager,
			deps.App.Configurator(),
			&deps.App.PublicKeepers,
		)

		// Use the real, persisted module versions from state (ctx)
		//
		// The "module.Manager.RunMigrations" interprets an empty
		// "module.VersionMap" as meaning that "none of the modules exist yet",
		// so would try run `InitGenesis` for all of the modules, including
		// x/capability, and that's going to panic with something like
		// "panic: SetIndex requires index to not be set".
		fromVm module.VersionMap

		// The plan MUST have (Height, Name) to be valid.
		// The plan MUST NOT have (Time).
		// It is not an IBC upgrade, so we set "UpgradedClientState" to nil.
		plan = upgradetypes.Plan{
			Name:                upgrade.UpgradeName,
			Time:                time.Time{}, // Time "zero" == unset on purpose
			Height:              deps.Ctx().BlockHeight(),
			Info:                "Testing Upgrade " + upgrade.UpgradeName,
			UpgradedClientState: (*codec.Any)(nil),
		}
	)

	err := plan.ValidateBasic()
	if err != nil {
		return fmt.Errorf("invalid upgrade.Plan: %w", err)
	}

	fromVm = deps.App.UpgradeKeeper.GetModuleVersionMap(deps.Ctx())

	updatedVm, err := upgradeHandler(
		deps.Ctx(),
		plan,
		fromVm,
	)
	if err != nil {
		return err
	}
	deps.App.UpgradeKeeper.SetModuleVersionMap(deps.Ctx(), updatedVm)
	return nil
}
