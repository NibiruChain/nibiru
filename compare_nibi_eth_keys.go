package main

import (
	"fmt"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/crypto/hd"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func main() {
	//app.SetPrefixes(appconst.AccountAddressPrefix)
	a := evmtest.NewEthAccInfo()
	fmt.Println(a.EthAddr.String())
	fmt.Println(a.NibiruAddr.String())

	mnemonic := "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"

	// Default derivation path m/44'/60'/0'/0/0
	// Cosmos derivation path m/44/118/100'/0/0
	//derivationPath := sdk.GetConfig().GetFullBIP44Path() // Default: eth.BIP44HDPath
	derivationPath := eth.BIP44HDPath // Default: eth.BIP44HDPath
	fmt.Println("Derivation path:", derivationPath)

	// Private & Public Keys
	//ethPrivateKeyBytes, _ := hd.EthSecp256k1.Derive()(mnemonic, keyring.DefaultBIP39Passphrase, eth.BIP44HDPath)
	ethPrivateKeyBytes, _ := hd.EthSecp256k1.Derive()(mnemonic, keyring.DefaultBIP39Passphrase, derivationPath)
	ethPrivateKey := hd.EthSecp256k1.Generate()(ethPrivateKeyBytes)

	// Public key of type go-ethereum ethsecp256k1
	ethPublicKey := ethPrivateKey.PubKey()

	// Convert eth public key to cosmos secp256k (custom address converter)
	ethPublicKeyOfCosmosType := secp256k1.PubKey{Key: ethPublicKey.Bytes()}

	ethAddress := ethPublicKey.Address()
	ethCosmosAddress := ethPublicKeyOfCosmosType.Address()

	fmt.Println("Nibi addr A eth addr:", sdk.AccAddress(ethAddress.Bytes()).String())
	fmt.Println("Nibi addr 2 cosmos addr:", sdk.AccAddress(ethCosmosAddress.Bytes()).String())

	fmt.Println("Hex 1:", "0x"+eth.BytesToHex(ethAddress.Bytes()))
	fmt.Println("Hex 2:", "0x"+eth.BytesToHex(ethCosmosAddress.Bytes()))

}
