package upgrades

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUpgrade2_20_0Registration verifies that v2.20 uses the standard handler
// and has no store migration. The binary's app wiring activates the Wasm guard.
func TestUpgrade2_20_0Registration(t *testing.T) {
	require.Equal(t, "v2.20.0", Upgrade2_20_0.UpgradeName)
	require.IsType(t, DefaultUpgradeHandler{}, Upgrade2_20_0.Handler)
	require.Empty(t, Upgrade2_20_0.StoreUpgrades.Added)
	require.Empty(t, Upgrade2_20_0.StoreUpgrades.Renamed)
	require.Empty(t, Upgrade2_20_0.StoreUpgrades.Deleted)

	found := false
	for _, upgrade := range AllUpgrades {
		if upgrade.UpgradeName == Upgrade2_20_0.UpgradeName {
			found = true
			break
		}
	}
	require.True(t, found, "v2.20.0 must be registered in AllUpgrades")
}
