package main

import (
	"fmt"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/appconst"
	"github.com/NibiruChain/nibiru/gosdk"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func main1() {
	app.SetPrefixes(appconst.AccountAddressPrefix)
	mnemonic := "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
	encCfg := app.MakeEncodingConfig()
	kr := keyring.NewInMemory(encCfg.Codec)
	privKey, addr, err := gosdk.PrivKeyFromMnemonic(kr, mnemonic, "test")
	if err != nil {
		panic(err)
	}
	pubKey := privKey.PubKey()
	addrStr := addr.String()
	fmt.Println("Address (expected):", addrStr)
	fmt.Println("Address from the public key (wrong):", sdk.AccAddress(pubKey.Address().Bytes()).String())
}
