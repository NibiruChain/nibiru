package upgrades

import (
	"fmt"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

var Upgrade2_21_0 = Upgrade{UpgradeName: "v2.21.0", Handler: Handler_v2_21{}}

type Handler_v2_21 struct{}

// Reviewed Sai operator snapshot. Seed on every chain running this upgrade.
var wasmDeployerSeed = []string{
	"nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l", // ud-prod
	"nibi1ss0s7fmw8n8t093mqt5was5c76k9amu0d7a5u0", // ud-prod-2
	"nibi1ljhfmddrxt3axx2y0f5dvt0mxhxkkvs3pewdmq", // Matthias
	"nibi1372pyz4cctz4ns434gdt82qc46a0eh48jqprr7", // Oleg
	"nibi137c35e2mdjucjzzu3vs6kpzwcxtgt9pdnvj5ll", // sai-perp-admin
}

func (h Handler_v2_21) Handler(mm *module.Manager, cfg module.Configurator, nibiru *keepers.PublicKeepers) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versions, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		state, err := nibiru.SudoKeeper.Sudoers.Get(ctx)
		if err != nil {
			return nil, err
		}
		_, err = nibiru.SudoKeeper.UpdateRoleMembers(sdk.WrapSDKContext(ctx), &sudo.MsgUpdateRoleMembers{
			Sender: state.Root, Role: sudo.RoleWasmDeployer, Add: wasmDeployerSeed,
		})
		if err != nil {
			return nil, fmt.Errorf("seed Wasm deployers: %w", err)
		}
		recoverErisV221(ctx, nibiru)
		return versions, nil
	}
}
