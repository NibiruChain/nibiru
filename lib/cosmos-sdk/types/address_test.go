package types_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/ed25519"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/bech32/legacybech32"
)

type addressTestSuite struct {
	suite.Suite
}

func TestAddressTestSuite(t *testing.T) {
	suite.Run(t, new(addressTestSuite))
}

func (s *addressTestSuite) SetupSuite() {
	s.T().Parallel()
}

var invalidStrs = []string{
	"hello, world!",
	"0xAA",
	"AAA",
	sdk.Bech32PrefixAccAddr + "AB0C",
	sdk.Bech32PrefixAccPub + "1234",
	sdk.Bech32PrefixValAddr + "5678",
	sdk.Bech32PrefixValPub + "BBAB",
	sdk.Bech32PrefixConsAddr + "FF04",
	sdk.Bech32PrefixConsPub + "6789",
}

func (s *addressTestSuite) testMarshal(original interface{}, res interface{}, marshal func() ([]byte, error), unmarshal func([]byte) error) {
	bz, err := marshal()
	s.Require().Nil(err)
	s.Require().Nil(unmarshal(bz))
	s.Require().Equal(original, res)
}

func (s *addressTestSuite) TestEmptyAddresses() {
	s.T().Parallel()
	s.Require().Equal((sdk.AccAddress{}).String(), "")
	s.Require().Equal((sdk.ValAddress{}).String(), "")
	s.Require().Equal((sdk.ConsAddress{}).String(), "")

	accAddr, err := sdk.AccAddressFromBech32("")
	s.Require().True(accAddr.Empty())
	s.Require().Error(err)

	valAddr, err := sdk.ValAddressFromBech32("")
	s.Require().True(valAddr.Empty())
	s.Require().Error(err)

	consAddr, err := sdk.ConsAddressFromBech32("")
	s.Require().True(consAddr.Empty())
	s.Require().Error(err)
}

func (s *addressTestSuite) TestYAMLMarshalers() {
	addr := secp256k1.GenPrivKey().PubKey().Address()

	acc := sdk.AccAddress(addr)
	val := sdk.ValAddress(addr)
	cons := sdk.ConsAddress(addr)

	got, _ := yaml.Marshal(&acc)
	s.Require().Equal(acc.String()+"\n", string(got))

	got, _ = yaml.Marshal(&val)
	s.Require().Equal(val.String()+"\n", string(got))

	got, _ = yaml.Marshal(&cons)
	s.Require().Equal(cons.String()+"\n", string(got))
}

func (s *addressTestSuite) TestRandBech32AccAddrConsistency() {
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}

	for i := 0; i < 1000; i++ {
		rand.Read(pub.Key)

		acc := sdk.AccAddress(pub.Address())
		res := sdk.AccAddress{}

		s.testMarshal(&acc, &res, acc.MarshalJSON, (&res).UnmarshalJSON)
		s.testMarshal(&acc, &res, acc.Marshal, (&res).Unmarshal)

		str := acc.String()
		res, err := sdk.AccAddressFromBech32(str)
		s.Require().Nil(err)
		s.Require().Equal(acc, res)

		str = hex.EncodeToString(acc)
		res, err = sdk.AccAddressFromHexUnsafe(str)
		s.Require().Nil(err)
		s.Require().Equal(acc, res)
	}

	for _, str := range invalidStrs {
		_, err := sdk.AccAddressFromHexUnsafe(str)
		s.Require().NotNil(err)

		_, err = sdk.AccAddressFromBech32(str)
		s.Require().NotNil(err)

		err = (*sdk.AccAddress)(nil).UnmarshalJSON([]byte("\"" + str + "\""))
		s.Require().NotNil(err)
	}

	_, err := sdk.AccAddressFromHexUnsafe("")
	s.Require().Equal(sdk.ErrEmptyHexAddress, err)
}

// Test that the account address cache ignores the bech32 prefix setting, retrieving bech32 addresses from the cache.
// This will cause the AccAddress.String() to print out unexpected prefixes if the config was changed between bech32 lookups.
// See https://github.com/cosmos/cosmos-sdk/issues/15317.
func (s *addressTestSuite) TestAddrCache() {
	// Use a random key
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}
	rand.Read(pub.Key)

	// Set SDK bech32 prefixes to 'osmo'
	prefix := "osmo"
	conf := sdk.GetConfig()
	conf.SetBech32PrefixForAccount(prefix, prefix+"pub")
	conf.SetBech32PrefixForValidator(prefix+"valoper", prefix+"valoperpub")
	conf.SetBech32PrefixForConsensusNode(prefix+"valcons", prefix+"valconspub")

	acc := sdk.AccAddress(pub.Address())
	osmoAddrBech32 := acc.String()

	// Set SDK bech32 to 'cosmos'
	prefix = "cosmos"
	conf.SetBech32PrefixForAccount(prefix, prefix+"pub")
	conf.SetBech32PrefixForValidator(prefix+"valoper", prefix+"valoperpub")
	conf.SetBech32PrefixForConsensusNode(prefix+"valcons", prefix+"valconspub")

	// We name this 'addrCosmos' to prove a point, but the bech32 address will still begin with 'osmo' due to the cache behavior.
	addrCosmos := sdk.AccAddress(pub.Address())
	cosmosAddrBech32 := addrCosmos.String()

	// The default behavior will retrieve the bech32 address from the cache, ignoring the bech32 prefix change.
	s.Require().Equal(osmoAddrBech32, cosmosAddrBech32)
	s.Require().True(strings.HasPrefix(osmoAddrBech32, "osmo"))
	s.Require().True(strings.HasPrefix(cosmosAddrBech32, "osmo"))
}

// Test that the bech32 prefix is respected when the address cache is disabled.
// This causes AccAddress.String() to print out the expected prefixes if the config is changed between bech32 lookups.
// See https://github.com/cosmos/cosmos-sdk/issues/15317.
func (s *addressTestSuite) TestAddrCacheDisabled() {
	sdk.SetAddrCacheEnabled(false)

	// Use a random key
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}
	rand.Read(pub.Key)

	// Set SDK bech32 prefixes to 'osmo'
	prefix := "osmo"
	conf := sdk.GetConfig()
	conf.SetBech32PrefixForAccount(prefix, prefix+"pub")
	conf.SetBech32PrefixForValidator(prefix+"valoper", prefix+"valoperpub")
	conf.SetBech32PrefixForConsensusNode(prefix+"valcons", prefix+"valconspub")

	acc := sdk.AccAddress(pub.Address())
	osmoAddrBech32 := acc.String()

	// Set SDK bech32 to 'cosmos'
	prefix = "cosmos"
	conf.SetBech32PrefixForAccount(prefix, prefix+"pub")
	conf.SetBech32PrefixForValidator(prefix+"valoper", prefix+"valoperpub")
	conf.SetBech32PrefixForConsensusNode(prefix+"valcons", prefix+"valconspub")

	addrCosmos := sdk.AccAddress(pub.Address())
	cosmosAddrBech32 := addrCosmos.String()

	// retrieve the bech32 address from the cache, respecting the bech32 prefix change.
	s.Require().NotEqual(osmoAddrBech32, cosmosAddrBech32)
	s.Require().True(strings.HasPrefix(osmoAddrBech32, "osmo"))
	s.Require().True(strings.HasPrefix(cosmosAddrBech32, "cosmos"))
}

func (s *addressTestSuite) TestValAddr() {
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}

	for i := 0; i < 20; i++ {
		rand.Read(pub.Key)

		acc := sdk.ValAddress(pub.Address())
		res := sdk.ValAddress{}

		s.testMarshal(&acc, &res, acc.MarshalJSON, (&res).UnmarshalJSON)
		s.testMarshal(&acc, &res, acc.Marshal, (&res).Unmarshal)

		str := acc.String()
		res, err := sdk.ValAddressFromBech32(str)
		s.Require().Nil(err)
		s.Require().Equal(acc, res)

		str = hex.EncodeToString(acc)
		res, err = sdk.ValAddressFromHex(str)
		s.Require().Nil(err)
		s.Require().Equal(acc, res)
	}

	for _, str := range invalidStrs {
		_, err := sdk.ValAddressFromHex(str)
		s.Require().NotNil(err)

		_, err = sdk.ValAddressFromBech32(str)
		s.Require().NotNil(err)

		err = (*sdk.ValAddress)(nil).UnmarshalJSON([]byte("\"" + str + "\""))
		s.Require().NotNil(err)
	}

	// test empty string
	_, err := sdk.ValAddressFromHex("")
	s.Require().Equal(sdk.ErrEmptyHexAddress, err)
}

func (s *addressTestSuite) TestConsAddress() {
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}

	for i := 0; i < 20; i++ {
		rand.Read(pub.Key[:])

		acc := sdk.ConsAddress(pub.Address())
		res := sdk.ConsAddress{}

		s.testMarshal(&acc, &res, acc.MarshalJSON, (&res).UnmarshalJSON)
		s.testMarshal(&acc, &res, acc.Marshal, (&res).Unmarshal)

		str := acc.String()
		res, err := sdk.ConsAddressFromBech32(str)
		s.Require().Nil(err)
		s.Require().Equal(acc, res)

		str = hex.EncodeToString(acc)
		res, err = sdk.ConsAddressFromHex(str)
		s.Require().Nil(err)
		s.Require().Equal(acc, res)
	}

	for _, str := range invalidStrs {
		_, err := sdk.ConsAddressFromHex(str)
		s.Require().NotNil(err)

		_, err = sdk.ConsAddressFromBech32(str)
		s.Require().NotNil(err)

		err = (*sdk.ConsAddress)(nil).UnmarshalJSON([]byte("\"" + str + "\""))
		s.Require().NotNil(err)
	}

	// test empty string
	_, err := sdk.ConsAddressFromHex("")
	s.Require().Equal(sdk.ErrEmptyHexAddress, err)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (s *addressTestSuite) TestConfiguredPrefix() {
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}
	for length := 1; length < 10; length++ {
		for times := 1; times < 20; times++ {
			rand.Read(pub.Key[:])
			// Test if randomly generated prefix of a given length works
			prefix := RandString(length)

			// Assuming that GetConfig is not sealed.
			config := sdk.GetConfig()
			config.SetBech32PrefixForAccount(
				prefix+sdk.PrefixAccount,
				prefix+sdk.PrefixPublic)

			acc := sdk.AccAddress(pub.Address())
			s.Require().True(strings.HasPrefix(
				acc.String(),
				prefix+sdk.PrefixAccount), acc.String())

			bech32Pub := legacybech32.MustMarshalPubKey(legacybech32.AccPK, pub)
			s.Require().True(strings.HasPrefix(
				bech32Pub,
				prefix+sdk.PrefixPublic))

			config.SetBech32PrefixForValidator(
				prefix+sdk.PrefixValidator+sdk.PrefixAddress,
				prefix+sdk.PrefixValidator+sdk.PrefixPublic)

			val := sdk.ValAddress(pub.Address())
			s.Require().True(strings.HasPrefix(
				val.String(),
				prefix+sdk.PrefixValidator+sdk.PrefixAddress))

			bech32ValPub := legacybech32.MustMarshalPubKey(legacybech32.ValPK, pub)
			s.Require().True(strings.HasPrefix(
				bech32ValPub,
				prefix+sdk.PrefixValidator+sdk.PrefixPublic))

			config.SetBech32PrefixForConsensusNode(
				prefix+sdk.PrefixConsensus+sdk.PrefixAddress,
				prefix+sdk.PrefixConsensus+sdk.PrefixPublic)

			cons := sdk.ConsAddress(pub.Address())
			s.Require().True(strings.HasPrefix(
				cons.String(),
				prefix+sdk.PrefixConsensus+sdk.PrefixAddress))

			bech32ConsPub := legacybech32.MustMarshalPubKey(legacybech32.ConsPK, pub)
			s.Require().True(strings.HasPrefix(
				bech32ConsPub,
				prefix+sdk.PrefixConsensus+sdk.PrefixPublic))
		}
	}
}

func (s *addressTestSuite) TestAddressInterface() {
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}
	rand.Read(pub.Key)

	addrs := []sdk.Address{
		sdk.ConsAddress(pub.Address()),
		sdk.ValAddress(pub.Address()),
		sdk.AccAddress(pub.Address()),
	}

	for _, addr := range addrs {
		switch addr := addr.(type) {
		case sdk.AccAddress:
			_, err := sdk.AccAddressFromBech32(addr.String())
			s.Require().Nil(err)
		case sdk.ValAddress:
			_, err := sdk.ValAddressFromBech32(addr.String())
			s.Require().Nil(err)
		case sdk.ConsAddress:
			_, err := sdk.ConsAddressFromBech32(addr.String())
			s.Require().Nil(err)
		default:
			s.T().Fail()
		}
	}
}

func (s *addressTestSuite) TestVerifyAddressFormat() {
	addr0 := make([]byte, 0)
	addr5 := make([]byte, 5)
	addr20 := make([]byte, 20)
	addr32 := make([]byte, 32)
	addr256 := make([]byte, 256)

	err := sdk.VerifyAddressFormat(addr0)
	s.Require().EqualError(err, "addresses cannot be empty: unknown address")
	err = sdk.VerifyAddressFormat(addr5)
	s.Require().NoError(err)
	err = sdk.VerifyAddressFormat(addr20)
	s.Require().NoError(err)
	err = sdk.VerifyAddressFormat(addr32)
	s.Require().NoError(err)
	err = sdk.VerifyAddressFormat(addr256)
	s.Require().EqualError(err, "address max length is 255, got 256: unknown address")
}

func (s *addressTestSuite) TestCustomAddressVerifier() {
	// Create a 10 byte address
	addr := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	accBech := sdk.AccAddress(addr).String()
	valBech := sdk.ValAddress(addr).String()
	consBech := sdk.ConsAddress(addr).String()
	// Verify that the default logic doesn't reject this 10 byte address
	// The default verifier is nil, we're only checking address length is
	// between 1-255 bytes.
	err := sdk.VerifyAddressFormat(addr)
	s.Require().Nil(err)
	_, err = sdk.AccAddressFromBech32(accBech)
	s.Require().Nil(err)
	_, err = sdk.ValAddressFromBech32(valBech)
	s.Require().Nil(err)
	_, err = sdk.ConsAddressFromBech32(consBech)
	s.Require().Nil(err)

	// Set a custom address verifier only accepts 20 byte addresses
	sdk.GetConfig().SetAddressVerifier(func(bz []byte) error {
		n := len(bz)
		if n == 20 {
			return nil
		}
		return fmt.Errorf("incorrect address length %d", n)
	})

	// Verifiy that the custom logic rejects this 10 byte address
	err = sdk.VerifyAddressFormat(addr)
	s.Require().NotNil(err)
	_, err = sdk.AccAddressFromBech32(accBech)
	s.Require().NotNil(err)
	_, err = sdk.ValAddressFromBech32(valBech)
	s.Require().NotNil(err)
	_, err = sdk.ConsAddressFromBech32(consBech)
	s.Require().NotNil(err)

	// Reinitialize the global config to default address verifier (nil)
	sdk.GetConfig().SetAddressVerifier(nil)
}

func (s *addressTestSuite) TestBech32ifyAddressBytes() {
	addr10byte := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	addr20byte := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	type args struct {
		prefix string
		bs     []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"empty address", args{"prefixa", []byte{}}, "", false},
		{"empty prefix", args{"", addr20byte}, "", true},
		{"10-byte address", args{"prefixa", addr10byte}, "prefixa1qqqsyqcyq5rqwzqf3953cc", false},
		{"10-byte address", args{"prefixb", addr10byte}, "prefixb1qqqsyqcyq5rqwzqf20xxpc", false},
		{"20-byte address", args{"prefixa", addr20byte}, "prefixa1qqqsyqcyq5rqwzqfpg9scrgwpugpzysn7hzdtn", false},
		{"20-byte address", args{"prefixb", addr20byte}, "prefixb1qqqsyqcyq5rqwzqfpg9scrgwpugpzysnrujsuw", false},
	}
	for _, tt := range tests {
		tt := tt
		s.T().Run(tt.name, func(t *testing.T) {
			got, err := sdk.Bech32ifyAddressBytes(tt.args.prefix, tt.args.bs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bech32ifyBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func (s *addressTestSuite) TestMustBech32ifyAddressBytes() {
	addr10byte := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	addr20byte := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	type args struct {
		prefix string
		bs     []byte
	}
	tests := []struct {
		name      string
		args      args
		want      string
		wantPanic bool
	}{
		{"empty address", args{"prefixa", []byte{}}, "", false},
		{"empty prefix", args{"", addr20byte}, "", true},
		{"10-byte address", args{"prefixa", addr10byte}, "prefixa1qqqsyqcyq5rqwzqf3953cc", false},
		{"10-byte address", args{"prefixb", addr10byte}, "prefixb1qqqsyqcyq5rqwzqf20xxpc", false},
		{"20-byte address", args{"prefixa", addr20byte}, "prefixa1qqqsyqcyq5rqwzqfpg9scrgwpugpzysn7hzdtn", false},
		{"20-byte address", args{"prefixb", addr20byte}, "prefixb1qqqsyqcyq5rqwzqfpg9scrgwpugpzysnrujsuw", false},
	}
	for _, tt := range tests {
		tt := tt
		s.T().Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				require.Panics(t, func() { sdk.MustBech32ifyAddressBytes(tt.args.prefix, tt.args.bs) })
				return
			}
			require.Equal(t, tt.want, sdk.MustBech32ifyAddressBytes(tt.args.prefix, tt.args.bs))
		})
	}
}

func (s *addressTestSuite) TestAddressTypesEquals() {
	addr1 := secp256k1.GenPrivKey().PubKey().Address()
	accAddr1 := sdk.AccAddress(addr1)
	consAddr1 := sdk.ConsAddress(addr1)
	valAddr1 := sdk.ValAddress(addr1)

	addr2 := secp256k1.GenPrivKey().PubKey().Address()
	accAddr2 := sdk.AccAddress(addr2)
	consAddr2 := sdk.ConsAddress(addr2)
	valAddr2 := sdk.ValAddress(addr2)

	// equality
	s.Require().True(accAddr1.Equals(accAddr1))
	s.Require().True(consAddr1.Equals(consAddr1))
	s.Require().True(valAddr1.Equals(valAddr1))

	// emptiness
	s.Require().True(sdk.AccAddress{}.Equals(sdk.AccAddress{}))
	s.Require().True(sdk.AccAddress{}.Equals(sdk.AccAddress(nil)))
	s.Require().True(sdk.AccAddress(nil).Equals(sdk.AccAddress{}))
	s.Require().True(sdk.AccAddress(nil).Equals(sdk.AccAddress(nil)))

	s.Require().True(sdk.ConsAddress{}.Equals(sdk.ConsAddress{}))
	s.Require().True(sdk.ConsAddress{}.Equals(sdk.ConsAddress(nil)))
	s.Require().True(sdk.ConsAddress(nil).Equals(sdk.ConsAddress{}))
	s.Require().True(sdk.ConsAddress(nil).Equals(sdk.ConsAddress(nil)))

	s.Require().True(sdk.ValAddress{}.Equals(sdk.ValAddress{}))
	s.Require().True(sdk.ValAddress{}.Equals(sdk.ValAddress(nil)))
	s.Require().True(sdk.ValAddress(nil).Equals(sdk.ValAddress{}))
	s.Require().True(sdk.ValAddress(nil).Equals(sdk.ValAddress(nil)))

	s.Require().False(accAddr1.Equals(accAddr2))
	s.Require().Equal(accAddr1.Equals(accAddr2), accAddr2.Equals(accAddr1))
	s.Require().False(consAddr1.Equals(consAddr2))
	s.Require().Equal(consAddr1.Equals(consAddr2), consAddr2.Equals(consAddr1))
	s.Require().False(valAddr1.Equals(valAddr2))
	s.Require().Equal(valAddr1.Equals(valAddr2), valAddr2.Equals(valAddr1))
}

func (s *addressTestSuite) TestNilAddressTypesEmpty() {
	s.Require().True(sdk.AccAddress(nil).Empty())
	s.Require().True(sdk.ConsAddress(nil).Empty())
	s.Require().True(sdk.ValAddress(nil).Empty())
}

func (s *addressTestSuite) TestGetConsAddress() {
	pk := secp256k1.GenPrivKey().PubKey()
	s.Require().NotEqual(sdk.GetConsAddress(pk), pk.Address())
	s.Require().True(bytes.Equal(sdk.GetConsAddress(pk).Bytes(), pk.Address().Bytes()))
	s.Require().Panics(func() { sdk.GetConsAddress(cryptotypes.PubKey(nil)) })
}

func (s *addressTestSuite) TestGetFromBech32() {
	_, err := sdk.GetFromBech32("", "prefix")
	s.Require().Error(err)
	s.Require().Equal("decoding Bech32 address failed: must provide a non empty address", err.Error())
	_, err = sdk.GetFromBech32("cosmos1qqqsyqcyq5rqwzqfys8f67", "x")
	s.Require().Error(err)
	s.Require().Equal("invalid Bech32 prefix; expected x, got cosmos", err.Error())
}
