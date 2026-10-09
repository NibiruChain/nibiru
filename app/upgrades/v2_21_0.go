package upgrades

import (
	"errors"
	"fmt"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

var Upgrade2_21_0 = Upgrade{UpgradeName: "v2.21.0", Handler: Handler_v2_21{}}

// Handler_v2_21 runs required schema migrations before independent custom steps.
// A failed custom step must leave the migrated chain usable and report its failure.
type Handler_v2_21 struct{}

// upgradeStepGasLimitV221 gives each custom step its own budget so exhaustion
// cannot consume the parent context's gas and prevent later steps from running.
const upgradeStepGasLimitV221 = uint64(10_000_000)

// Reviewed Sai operator snapshot. Seed on every chain running this upgrade.
var wasmDeployerSeed = []string{
	"nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l", // ud-prod
	"nibi1ss0s7fmw8n8t093mqt5was5c76k9amu0d7a5u0", // ud-prod-2
	"nibi1ljhfmddrxt3axx2y0f5dvt0mxhxkkvs3pewdmq", // Matthias
	"nibi1372pyz4cctz4ns434gdt82qc46a0eh48jqprr7", // Oleg
	"nibi137c35e2mdjucjzzu3vs6kpzwcxtgt9pdnvj5ll", // sai-perp-admin
}

// Handler returns migration errors because the binary requires the migrated
// schema. Custom deployment grants and incident recovery can fail independently
// without stopping the upgrade. The outer handler reports their failures on the
// parent context after their caches are discarded.
func (h Handler_v2_21) Handler(mm *module.Manager, cfg module.Configurator, nibiru *keepers.PublicKeepers) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versions, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		if err := h.runUpgrade2_21_0(ctx, nibiru); err != nil {
			ctx.Logger().Error("v2.21.0 upgrade failure", "err", err)
			ctx.EventManager().EmitEvent(NewEventUpgradeFailure("v2.21.0", err))
		}
		return versions, nil
	}
}

// runUpgrade2_21_0 contains custom upgrade failures without undoing successful
// migrations. Each step commits independently; a failed grant must not prevent
// recovery, and a failed recovery must not discard completed deployment grants.
func (h Handler_v2_21) runUpgrade2_21_0(ctx sdk.Context, nibiru *keepers.PublicKeepers) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = errors.Join(err, h.upgradePanicError("custom upgrade", value))
		}
	}()
	if seedErr := h.runCachedUpgradeStep(ctx, func(cached sdk.Context) error {
		return h.seedWasmDeployers(cached, nibiru)
	}); seedErr != nil {
		err = fmt.Errorf("seed Wasm deployers: %w", seedErr)
	}
	return errors.Join(err, h.recoverErisV221(ctx, nibiru))
}

// runCachedUpgradeStep commits a custom step only when it returns successfully.
// The recovery boundary includes callback execution and cache commit. Failure
// events belong on the caller's context so discarding a cache preserves the report.
func (h Handler_v2_21) runCachedUpgradeStep(ctx sdk.Context, run func(sdk.Context) error) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = h.upgradePanicError("cached upgrade step", value)
		}
	}()
	cached, commit := ctx.CacheContext()
	cached = cached.WithGasMeter(sdk.NewGasMeter(upgradeStepGasLimitV221))
	if err := run(cached); err != nil {
		return err
	}
	commit()
	return nil
}

// seedWasmDeployers grants the reviewed operator set through the normal sudo
// keeper after migration. Its caller isolates writes so an invalid seed or
// damaged sudo state cannot leave a partially applied deployment role.
func (h Handler_v2_21) seedWasmDeployers(ctx sdk.Context, nibiru *keepers.PublicKeepers) error {
	state, err := nibiru.SudoKeeper.Sudoers.Get(ctx)
	if err != nil {
		return err
	}
	_, err = nibiru.SudoKeeper.UpdateRoleMembers(sdk.WrapSDKContext(ctx), &sudo.MsgUpdateRoleMembers{
		Sender: state.Root, Role: sudo.RoleWasmDeployer, Add: wasmDeployerSeed,
	})
	return err
}

// upgradePanicError converts recoverable Go panics to the same reporting path as
// returned errors. Fatal process faults remain outside Go panic recovery.
func (h Handler_v2_21) upgradePanicError(step string, value any) error {
	if gas, ok := value.(sdk.ErrorOutOfGas); ok {
		return fmt.Errorf("%s out of gas: %s", step, gas.Descriptor)
	}
	return fmt.Errorf("%s panic (%T): %v", step, value, value)
}
