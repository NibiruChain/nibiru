package upgrades

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUpgrade2_23_0Registration verifies that v2.23 uses the standard handler,
// has no store migration, and is registered in AllUpgrades, so its plan runs
// in the binary that also enforces the lane rules keyed on it.
func TestUpgrade2_23_0Registration(t *testing.T) {
	require.Equal(t, "v2.23.0", Upgrade2_23_0.UpgradeName)
	require.IsType(t, DefaultUpgradeHandler{}, Upgrade2_23_0.Handler)
	require.Empty(t, Upgrade2_23_0.StoreUpgrades.Added)
	require.Empty(t, Upgrade2_23_0.StoreUpgrades.Renamed)
	require.Empty(t, Upgrade2_23_0.StoreUpgrades.Deleted)
	require.Equal(t, Upgrade2_23_0.UpgradeName,
		AllUpgrades[len(AllUpgrades)-1].UpgradeName,
		"v2.23.0 must be the latest entry in AllUpgrades")
}
