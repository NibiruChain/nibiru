package sample

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// Returns a sample account address (sdk.AccAddress)
// Note that AccAddress().String() can be used to get a string representation.
func AccAddress() sdk.AccAddress {
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr)
}

// PrivKeyAddressPairsFromRand generates (deterministically) a total of n private keys and addresses.
func PrivKeyAddressPairs(n int) (keys []cryptotypes.PrivKey, addrs []sdk.AccAddress) {
	r := rand.New(rand.NewSource(12345)) // make the generation deterministic
	keys = make([]cryptotypes.PrivKey, n)
	addrs = make([]sdk.AccAddress, n)
	for i := 0; i < n; i++ {
		secret := make([]byte, 32)
		_, err := r.Read(secret)
		if err != nil {
			panic("Could not read randomness")
		}
		keys[i] = ed25519.GenPrivKeyFromSecret(secret)
		addrs[i] = sdk.AccAddress(keys[i].PubKey().Address())
	}
	return
}
