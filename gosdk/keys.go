package gosdk

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/codec"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
)

func EncodingConfig() codec.EncodingConfig { return app.MakeEncodingConfig() }

// NewKeyring: Creates an empty, in-memory keyring
func NewKeyring() keyring.Keyring {
	return keyring.NewInMemory(EncodingConfig().Codec)
}

// TODO: Is it necessary to add support for interacting with local file system
// keyring?
// import (
//   "bufio"
//   "os"
//   "path/filepath"
// )
// func NewKeyringLocal(nodeDir string) (keyring.Keyring) {
//     clientDir := filepath.Join(nodeDir, "keyring")
// 	var cdc codec.Codec = EncodingConfig.Marshaler
// 	buf := bufio.NewReader(os.Stdin)
// 	return keyring.New(
// 		sdk.KeyringServiceName(),
// 		keyring.BackendTest,
// 		clientDir,
// 		buf,
// 		cdc,
// 	)
// }

func AddSignerToKeyringSecp256k1(
	kring keyring.Keyring, mnemonic string, keyName string,
) (sdk.AccAddress, error) {
	algo := hd.Secp256k1
	overwrite := true
	addr, secretMnem, err := sdktestutil.GenerateSaveCoinKey(
		kring, keyName, mnemonic, overwrite, algo,
	)
	if err != nil {
		return nil, fmt.Errorf("%w : Failed Key Generation with mnemonic %s", err, secretMnem)
	}

	return addr, err
}
