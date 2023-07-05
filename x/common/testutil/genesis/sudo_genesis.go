package genesis

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"

	"github.com/NibiruChain/nibiru/x/sudo"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/NibiruChain/nibiru/x/common/testutil"
)

func AddSudoGenesis(gen app.GenesisState) (
	genState app.GenesisState,
	rootPrivKey cryptotypes.PrivKey,
	rootAddr sdk.AccAddress,
) {
	sudoGenesis, rootPrivKey, rootAddr := SudoGenesis()
	gen[sudotypes.ModuleName] = TEST_ENCODING_CONFIG.Marshaler.
		MustMarshalJSON(sudoGenesis)
	return gen, rootPrivKey, rootAddr
}

func SudoGenesis() (
	genState *sudotypes.GenesisState,
	rootPrivKey cryptotypes.PrivKey,
	rootAddr sdk.AccAddress,
) {
	sudoGenesis := sudo.DefaultGenesis()

	// Set the root user
	privKeys, addrs := testutil.PrivKeyAddressPairs(1)
	rootPrivKey = privKeys[0]
	rootAddr = addrs[0]
	sudoGenesis.Sudoers.Root = rootAddr.String()

	return sudoGenesis, rootPrivKey, rootAddr
}
