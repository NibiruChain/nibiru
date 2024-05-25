package gosdk

import (
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/codec"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
)

func EncodingConfig() codec.EncodingConfig { return app.MakeEncodingConfig() }

// NewKeyring: Creates an empty, in-memory keyring
func NewKeyring() keyring.Keyring {
	return keyring.NewInMemory(EncodingConfig().Codec)
}

// TODO: Is this needed?
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

func PrivKeyFromMnemonic(
	kring keyring.Keyring, mnemonic string, keyName string,
) (cryptotypes.PrivKey, sdk.AccAddress, error) {
	algo := hd.Secp256k1
	overwrite := true
	addr, secret, err := sdktestutil.GenerateSaveCoinKey(
		kring, keyName, mnemonic, overwrite, algo,
	)
	if err != nil {
		return &secp256k1.PrivKey{}, sdk.AccAddress{}, err
	}
	privKey := secp256k1.GenPrivKeyFromSecret([]byte(secret))
	return privKey, addr, err
}

func CreateSigner(
	mnemonic string,
	kring keyring.Keyring,
	keyName string,
) (kringRecord *keyring.Record, privKey cryptotypes.PrivKey, err error) {
	privKey, _, err = PrivKeyFromMnemonic(kring, mnemonic, keyName)
	if err != nil {
		return kringRecord, privKey, err
	}
	kringRecord, err = CreateSignerFromPrivKey(privKey, keyName)
	return kringRecord, privKey, err
}

func CreateSignerFromPrivKey(
	privKey cryptotypes.PrivKey, keyName string,
) (*keyring.Record, error) {
	return keyring.NewLocalRecord(keyName, privKey, privKey.PubKey())
}

func AddSignerToKeyring(
	kring keyring.Keyring, privKey cryptotypes.PrivKey, keyName string,
) error {
	passphrase := "password"
	armor := crypto.EncryptArmorPrivKey(privKey, passphrase, privKey.Type())
	return kring.ImportPrivKey(keyName, armor, passphrase)
}
