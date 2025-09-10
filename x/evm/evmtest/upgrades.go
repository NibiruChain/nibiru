package evmtest

import (
	"fmt"
	"time"

	codec "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

func (deps *TestDeps) RunUpgrade(upgrade upgrades.Upgrade) error {
	upgradeHandler := upgrade.CreateUpgradeHandler(
		deps.App.ModuleManager,
		module.NewConfigurator(
			deps.App.AppCodec(),
			deps.App.MsgServiceRouter(),
			deps.App.GRPCQueryRouter(),
		),
		&deps.App.PublicKeepers,
		deps.App.GetIBCKeeper().ClientKeeper,
	)

	// ---- Run the upgrade handler. ----

	var (
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
			Height:              deps.Ctx.BlockHeight(),
			Info:                "Testing Upgrade " + upgrade.UpgradeName,
			UpgradedClientState: (*codec.Any)(nil),
		}
	)

	err := plan.ValidateBasic()
	if err != nil {
		return fmt.Errorf("invalid upgrade.Plan: %w", err)
	}

	fromVm = deps.App.UpgradeKeeper.GetModuleVersionMap(deps.Ctx)

	_, err = upgradeHandler(
		deps.Ctx,
		plan,
		fromVm,
	)
	return err
}
