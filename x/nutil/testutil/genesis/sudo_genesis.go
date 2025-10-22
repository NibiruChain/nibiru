package genesis

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func AddSudoGenesis(gen app.GenesisState) (
	genState app.GenesisState,
	rootPrivKey cryptotypes.PrivKey,
	rootAddr sdk.AccAddress,
) {
	sudoGenesis, rootPrivKey, rootAddr := SudoGenesis()
	gen[sudo.ModuleName] = app.MakeEncodingConfig().Codec.MustMarshalJSON(sudoGenesis)
	return gen, rootPrivKey, rootAddr
}

func SudoGenesis() (
	genState *sudo.GenesisState,
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
