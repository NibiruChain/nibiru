package upgrades_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func TestUpgrade2_21_0MigratesAndSeedsEveryChain(t *testing.T) {
	for _, chainID := range []string{"cataclysm-1", "nibiru-testnet-3", "localnet"} {
		t.Run(chainID, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext()
			ctx = ctx.WithChainID(chainID)
			root, former := testutil.NewAccAddress(), testutil.NewAccAddress()
			hook := sdk.AccAddress(bytes.Repeat([]byte{8}, 32)).String()
			zeroGas := sudo.ZeroGasActors{Senders: []string{former.String()}, Contracts: []string{hook}}
			app.SudoKeeper.ZeroGasActors.Set(ctx, zeroGas)
			app.SudoKeeper.WasmBlockHooksContract.Set(ctx, hook)
			legacy := protowire.AppendTag(nil, 1, protowire.BytesType)
			legacy = protowire.AppendString(legacy, root.String())
			legacy = protowire.AppendTag(legacy, 2, protowire.BytesType)
			legacy = protowire.AppendString(legacy, former.String())
			key := make([]byte, 8)
			binary.BigEndian.PutUint64(key, 0)
			collections.Map[uint64, sudo.Sudoers](app.SudoKeeper.Sudoers).GetStore(ctx).Set(key, legacy)
			fromVM := app.ModuleManager.GetVersionMap()
			fromVM[sudo.ModuleName] = 1
			handler := upgrades.Upgrade2_21_0.Handler.Handler(app.ModuleManager, app.Configurator(), &app.PublicKeepers)
			versions, err := handler(ctx, upgradetypes.Plan{Name: "v2.21.0"}, fromVM)
			require.NoError(t, err)
			require.Equal(t, uint64(2), versions[sudo.ModuleName])
			state, err := app.SudoKeeper.Sudoers.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, root.String(), state.Root)
			require.Len(t, state.Roles, 3)
			require.Equal(t, sudo.RoleChainParams, state.Roles[0].Role)
			require.Equal(t, sudo.RoleTFOper, state.Roles[1].Role)
			require.Equal(t, []string{former.String()}, state.Roles[0].Members)
			require.Equal(t, []string{former.String()}, state.Roles[1].Members)
			require.Equal(t, sudo.RoleWasmDeployer, state.Roles[2].Role)
			require.ElementsMatch(t, []string{
				"nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l",
				"nibi1ss0s7fmw8n8t093mqt5was5c76k9amu0d7a5u0",
				"nibi1ljhfmddrxt3axx2y0f5dvt0mxhxkkvs3pewdmq",
				"nibi1372pyz4cctz4ns434gdt82qc46a0eh48jqprr7",
				"nibi137c35e2mdjucjzzu3vs6kpzwcxtgt9pdnvj5ll",
			}, state.Roles[2].Members)
			require.Error(t, app.SudoKeeper.CheckPermissions(former, ctx, sudo.RoleWasmDeployer))
			require.Equal(t, hook, app.SudoKeeper.WasmBlockHooksContract.GetOr(ctx, ""))
			require.Equal(t, zeroGas, app.SudoKeeper.ZeroGasActors.GetOr(ctx, sudo.ZeroGasActors{}))
			require.NoError(t, app.SudoKeeper.ExportGenesis(ctx).Validate())
			if chainID != "cataclysm-1" {
				for _, event := range ctx.EventManager().Events() {
					require.NotEqual(t, "eris_recovery", event.Type)
				}
			}
		})
	}
}
