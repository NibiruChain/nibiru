package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
)

// TestUpgrade2_23RecognizesTestnetHistory exercises the downgrade guard against
// testnet's completed release name, then executes the next named upgrade.
func TestUpgrade2_23RecognizesTestnetHistory(t *testing.T) {
	chain, ctx := testapp.NewNibiruTestAppAndContext()
	ctx = ctx.WithChainID("nibiru-testnet-2").WithBlockHeight(20)
	require.True(t, chain.UpgradeKeeper.HasHandler("v2.22.0"))
	require.True(t, chain.UpgradeKeeper.HasHandler("v2.23.0"))
	for _, release := range app.Upgrades {
		require.NotEqual(t, "v2.22.0", release.UpgradeName, "there is no separate v2.22 release implementation")
	}

	// Seed the completed record as the old testnet binary would have done.
	// This temporary callback is test setup, not another release handler.
	chain.UpgradeKeeper.SetUpgradeHandler("v2.22.0", func(_ sdk.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
		return versions, nil
	})
	chain.UpgradeKeeper.ApplyUpgrade(ctx, upgradetypes.Plan{Name: "v2.22.0", Height: 20})
	chain.UpgradeKeeper.SetUpgradeHandler("v2.22.0", upgrades.Upgrade2_23_0.Handler.Handler(chain.ModuleManager, chain.Configurator(), &chain.PublicKeepers))
	chain.UpgradeKeeper.SetDowngradeVerified(false)
	ctx = ctx.WithBlockHeight(21)
	require.NotPanics(t, func() {
		chain.BeginBlocker(ctx, abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	})
	require.Equal(t, int64(20), chain.UpgradeKeeper.GetDoneHeight(ctx, "v2.22.0"))

	require.NoError(t, chain.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: "v2.23.0", Height: 22}))
	ctx = ctx.WithBlockHeight(22)
	response := chain.BeginBlocker(ctx, abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	report, err := upgrades.ReconstructUSDCMigration(response.Events)
	require.NoError(t, err)
	require.Equal(t, "skipped", report.Status, "no USDC holder registry was predeployed")
	require.Equal(t, int64(22), chain.UpgradeKeeper.GetDoneHeight(ctx, "v2.23.0"))
	require.Equal(t, int64(20), chain.UpgradeKeeper.GetDoneHeight(ctx, "v2.22.0"))
}
