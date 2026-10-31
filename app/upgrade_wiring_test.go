package app_test

import (
	"path/filepath"
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
)

func TestLegacyUpgradeWiringOptions(t *testing.T) {
	home := t.TempDir()
	nibiru := app.NewNibiruApp(log.NewNopLogger(), tmdb.NewMemDB(), nil, true, sims.AppOptionsMap{
		flags.FlagHome:                home,
		server.FlagUnsafeSkipUpgrades: []int{12, 34},
	})

	require.True(t, nibiru.UpgradeKeeper.IsSkipHeight(12))
	require.True(t, nibiru.UpgradeKeeper.IsSkipHeight(34))
	require.False(t, nibiru.UpgradeKeeper.IsSkipHeight(13))
	infoPath, err := nibiru.UpgradeKeeper.GetUpgradeInfoPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "data", upgradetypes.UpgradeInfoFilename), infoPath)
	require.Same(t, nibiru.BaseApp, nibiru.UpgradeKeeper.GetVersionSetter())
}

func TestLegacyUpgradeWiringExecutesPlan(t *testing.T) {
	nibiru, ctx := testapp.NewNibiruTestAppAndContext()
	versions := nibiru.ModuleManager.GetVersionMap()
	require.Equal(t, versions, nibiru.UpgradeKeeper.GetModuleVersionMap(ctx))
	require.Equal(t, uint64(2), versions[upgradetypes.ModuleName])
	require.Equal(t, upgradetypes.ModuleName, nibiru.ModuleManager.OrderBeginBlockers[0])

	// Exercise the registered Msg service to preserve its governance authority
	// and its storage of a pending height-based plan.
	plan := upgradetypes.Plan{Name: "legacy-wiring-test", Height: ctx.BlockHeight()}
	handler := nibiru.MsgServiceRouter().Handler(&upgradetypes.MsgSoftwareUpgrade{})
	require.NotNil(t, handler)
	_, err := handler(ctx, &upgradetypes.MsgSoftwareUpgrade{
		Authority: authtypes.NewModuleAddress("not-governance").String(),
		Plan:      plan,
	})
	require.Error(t, err)
	_, found := nibiru.UpgradeKeeper.GetUpgradePlan(ctx)
	require.False(t, found)

	_, err = handler(ctx, &upgradetypes.MsgSoftwareUpgrade{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Plan:      plan,
	})
	require.NoError(t, err)

	before := nibiru.AppVersion()
	called := false
	nibiru.UpgradeKeeper.SetUpgradeHandler(plan.Name, func(_ sdk.Context, got upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		called = true
		require.Equal(t, plan, got)
		require.Equal(t, versions, fromVM)
		return fromVM, nil
	})
	nibiru.BeginBlocker(ctx, abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	require.True(t, called)
	require.Equal(t, before+1, nibiru.AppVersion())
	require.Equal(t, versions, nibiru.UpgradeKeeper.GetModuleVersionMap(ctx))
	_, found = nibiru.UpgradeKeeper.GetUpgradePlan(ctx)
	require.False(t, found)
	require.Equal(t, ctx.BlockHeight(), nibiru.UpgradeKeeper.GetDoneHeight(ctx, plan.Name))
}
