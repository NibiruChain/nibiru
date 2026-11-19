package upgrades_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	devgas "github.com/NibiruChain/nibiru/v2/x/devgas/v1"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

// TestUpgrade2_24_0ClearsDevgasState seeds the x/devgas state that mainnet
// has before the upgrade, registered fee shares with fee sharing enabled,
// and checks that the v2.24.0 handler deletes the fee shares, disables fee
// sharing, keeps the other params, and leaves the module version map as it
// was. A second run changes nothing.
func TestUpgrade2_24_0ClearsDevgasState(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	k := deps.App.DevGasKeeper

	params := devgastypes.DefaultParams()
	params.EnableFeeShare = true
	k.ModuleParams.Set(ctx, params)

	withdrawer := testutil.NewAccAddress().String()
	var contracts []sdk.AccAddress
	for range 12 {
		contract := testutil.NewAccAddress()
		contracts = append(contracts, contract)
		k.SetFeeShare(ctx, devgastypes.FeeShare{
			ContractAddress:   contract.String(),
			DeployerAddress:   testutil.NewAccAddress().String(),
			WithdrawerAddress: withdrawer,
		})
	}
	require.Len(t, devgas.ExportGenesis(ctx, k).FeeShare, len(contracts))
	require.True(t, k.GetParams(ctx).EnableFeeShare)
	versionsBefore := deps.App.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.Contains(t, versionsBefore, devgastypes.ModuleName)

	require.NoError(t, deps.RunUpgrade(upgrades.Upgrade2_24_0))
	ctx = deps.Ctx()

	for _, contract := range contracts {
		require.False(t, k.IsFeeShareRegistered(ctx, contract))
	}
	genesis := devgas.ExportGenesis(ctx, k)
	require.Empty(t, genesis.FeeShare)
	require.NoError(t, genesis.Validate())

	wantParams := params.Sanitize()
	wantParams.EnableFeeShare = false
	require.Equal(t, wantParams, k.GetParams(ctx))

	// The upgrade bumps no module consensus version, and x/devgas stays in
	// the version map because its store is not deleted.
	require.Equal(t, versionsBefore, deps.App.UpgradeKeeper.GetModuleVersionMap(ctx))

	require.NoError(t, deps.RunUpgrade(upgrades.Upgrade2_24_0))
	require.Empty(t, devgas.ExportGenesis(deps.Ctx(), k).FeeShare)
	require.Equal(t, wantParams, k.GetParams(deps.Ctx()))
	require.Equal(t, versionsBefore, deps.App.UpgradeKeeper.GetModuleVersionMap(deps.Ctx()))
}

// TestUpgrade2_24_0Registered checks that the app registers the handler
// under the plan name governance will use and that the upgrade changes no
// store keys.
func TestUpgrade2_24_0Registered(t *testing.T) {
	deps := evmtest.NewTestDeps()
	require.Equal(t, "v2.24.0", upgrades.Upgrade2_24_0.UpgradeName)
	require.True(t, deps.App.UpgradeKeeper.HasHandler("v2.24.0"))
	require.Empty(t, upgrades.Upgrade2_24_0.StoreUpgrades.Added)
	require.Empty(t, upgrades.Upgrade2_24_0.StoreUpgrades.Renamed)
	require.Empty(t, upgrades.Upgrade2_24_0.StoreUpgrades.Deleted)
}

// TestUpgrade2_24_0CleanupFailureFailsUpgrade checks that a failed x/devgas
// cleanup fails the upgrade and that the handler recovers no panic. Either
// outcome makes x/upgrade abort the block, so a node with a local store
// failure halts and does not continue with state that other nodes deleted.
func TestUpgrade2_24_0CleanupFailureFailsUpgrade(t *testing.T) {
	seed := func(deps *evmtest.TestDeps) devgastypes.FeeShare {
		feeShare := devgastypes.FeeShare{
			ContractAddress:   testutil.NewAccAddress().String(),
			DeployerAddress:   testutil.NewAccAddress().String(),
			WithdrawerAddress: testutil.NewAccAddress().String(),
		}
		deps.App.DevGasKeeper.SetFeeShare(deps.Ctx(), feeShare)
		return feeShare
	}

	t.Run("a returned error fails the upgrade", func(t *testing.T) {
		deps := evmtest.NewTestDeps()
		seed(&deps)
		errCleanup := errors.New("store read failed")
		upgrade := upgrades.Upgrade2_24_0
		upgrade.Handler = upgrades.Handler_v2_24{
			ClearDevgas: func(sdk.Context) (int, error) { return 0, errCleanup },
		}
		err := deps.RunUpgrade(upgrade)
		require.ErrorIs(t, err, errCleanup)
		require.ErrorContains(t, err, "clear x/devgas state")
	})

	t.Run("a panic propagates", func(t *testing.T) {
		deps := evmtest.NewTestDeps()
		seed(&deps)
		upgrade := upgrades.Upgrade2_24_0
		upgrade.Handler = upgrades.Handler_v2_24{
			ClearDevgas: func(sdk.Context) (int, error) { panic("iavl: database error") },
		}
		require.PanicsWithValue(t, "iavl: database error", func() {
			_ = deps.RunUpgrade(upgrade)
		})
	})
}
