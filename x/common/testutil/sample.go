package testutil

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

// AccAddress returns a sample address (sdk.AccAddress) created using secp256k1.
// Note that AccAddress().String() can be used to get a string representation.
func AccAddress() sdk.AccAddress {
	_, accAddr := PrivKey()
	return accAddr
}

// PrivKey returns a private key and corresponding on-chain address.
func PrivKey() (*secp256k1.PrivKey, sdk.AccAddress) {
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := pubKey.Address()
	return privKey, sdk.AccAddress(addr)
}

// PrivKeyAddressPairs generates (deterministically) a total of n private keys
// and addresses.
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
		keys[i] = secp256k1.GenPrivKeyFromSecret(secret)
		addrs[i] = sdk.AccAddress(keys[i].PubKey().Address())
	}
	return
}

func BlankContext(storeKeyName string) sdk.Context {
	storeKey := sdk.NewKVStoreKey(storeKeyName)
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, types.StoreTypeIAVL, db)
	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	return ctx
}

type TypeLatin struct {
	Letters    string
	CapLetters string
	Numbers    string
}

var Latin = TypeLatin{
	Letters:    "abcdefghijklmnopqrstuvwxyz",
	CapLetters: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	Numbers:    "0123456789",
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = Latin.Letters[rand.Intn(len(Latin.Letters))]
	}
	return string(b)
}
