package collections

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/gogo/protobuf/types"
)

type SuiteValueEncoder struct {
	suite.Suite
}

func TestSuiteValueEncoder_RunAll(t *testing.T) {
	suite.Run(t, new(SuiteValueEncoder))
}

func (s *SuiteValueEncoder) TestProtoValueEncoder() {
	s.T().Run("bijectivity", func(t *testing.T) {
		protoType := types.BytesValue{Value: []byte("testing")}

		registry := testdata.NewTestInterfaceRegistry()
		cdc := codec.NewProtoCodec(registry)

		assertValueBijective[types.BytesValue](t, ProtoValueEncoder[types.BytesValue](cdc), protoType)
	})
}

func (s *SuiteValueEncoder) TestDecValueEncoder() {
	s.Run("bijectivity", func() {
		assertValueBijective(s.T(), DecValueEncoder, sdk.MustNewDecFromStr("-1000.5858"))
	})
}

func (s *SuiteValueEncoder) TestAccAddressValueEncoder() {
	s.Run("bijectivity", func() {
		assertValueBijective(s.T(), AccAddressValueEncoder, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
	})
}

func (s *SuiteValueEncoder) TestUint64ValueEncoder() {
	s.Run("bijectivity", func() {
		assertValueBijective(s.T(), Uint64ValueEncoder, 1000)
	})
}

func (s *SuiteValueEncoder) TestIntEncoder() {
	// we test our assumptions around int are correct.
	outOfBounds := new(big.Int).Lsh(big.NewInt(1), 256)       // 2^256
	maxBigInt := new(big.Int).Sub(outOfBounds, big.NewInt(1)) // 2^256 - 1
	s.Equal(maxBigInt.BitLen(), math.MaxBitLen)
	s.Greater(outOfBounds.BitLen(), math.MaxBitLen)

	s.NotPanics(func() {
		sdk.NewIntFromBigInt(maxBigInt)
	})
	s.Panics(func() {
		sdk.NewIntFromBigInt(outOfBounds)
	})

	s.Require().Equal(maxIntKeyLen, len(maxBigInt.Bytes()))

	// test encoding ordering
	enc1 := IntKeyEncoder.Encode(sdk.NewInt(50_000))
	enc2 := IntKeyEncoder.Encode(sdk.NewInt(100_000))
	s.Less(enc1, enc2)

	// test decoding
	size, got1 := IntKeyEncoder.Decode(enc1)
	s.Equal(maxIntKeyLen, size)
	_, got2 := IntKeyEncoder.Decode(enc2)
	s.Equal(sdk.NewInt(50_000), got1)
	s.Equal(sdk.NewInt(100_000), got2)

	// require panics on negative values
	s.Panics(func() {
		IntKeyEncoder.Encode(sdk.NewInt(-1))
	})
	// require panics on invalid int
	s.Panics(func() {
		IntKeyEncoder.Encode(math.Int{})
	})

	// test value encoder
	value := sdk.NewInt(50_000)
	valueBytes := IntValueEncoder.Encode(value)
	gotValue := IntValueEncoder.Decode(valueBytes)
	s.Equal(value, gotValue)

	// panics on invalid math.Int
	s.Panics(func() {
		IntValueEncoder.Encode(math.Int{})
	})
}
