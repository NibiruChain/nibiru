//go:build nibiru_upgrade_e2e_next

package upgrades

// This handler exists only in the successor binary used by the isolated
// upgrade runner. Registering it in the preceding binary would trigger the
// upgrade module's early-binary guard before the scheduled height.
func init() {
	AllUpgrades = append(AllUpgrades, NewVanillaUpgrade("e2e-x-upgrade-next"))
}
