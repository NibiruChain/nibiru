package upgrades_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
)

// A fixture rebuild must also update the production profile. Tests using their
// own fixture profile otherwise could pass while the shipped handler looked at
// an address derived from obsolete bytecode.
func TestUpgrade2_23_0RegistryProfileMatchesFixture(t *testing.T) {
	_, checksum := holderSetBytecode(t)
	profile := upgrades.DefaultUSDCMigrationConfigV223()
	require.Equal(t, checksum, profile.RegistryChecksum)
	creator := sdk.MustAccAddressFromBech32(upgrades.USDCHolderSetCreatorV223)
	require.Equal(t, wasmkeeper.BuildContractAddressPredictable(checksum, creator,
		[]byte(upgrades.USDCHolderSetSaltV223), nil), profile.Registry)
}
