package collections

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/secp256k1"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/gogo/protobuf/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
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

func (s *SuiteValueEncoder) TestIntKeyEncoder() {
	// we test our assumptions around int are correct.
	outOfBounds := new(big.Int).Lsh(big.NewInt(1), 256)       // 2^256
	maxBigInt := new(big.Int).Sub(outOfBounds, big.NewInt(1)) // 2^256 - 1
	s.Equal(maxBigInt.BitLen(), sdkmath.MaxBitLen)
	s.Greater(outOfBounds.BitLen(), sdkmath.MaxBitLen)

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
		IntKeyEncoder.Encode(sdkmath.Int{})
	})
}

func (s *SuiteValueEncoder) TestIntValueEncoderSigned() {
	cases := []sdkmath.Int{
		sdkmath.NewInt(-1),
		sdkmath.NewInt(-50_000),
		sdkmath.ZeroInt(),
		sdkmath.NewInt(1),
		sdkmath.NewInt(50_000),
		sdkmath.NewIntFromBigInt(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))),
	}
	for _, value := range cases {
		s.Run(value.String(), func() {
			assertValueBijective(s.T(), IntValueEncoder, value)
		})
	}
}

func (s *SuiteValueEncoder) TestUintValueEncoderFixedWidth() {
	outOfBounds := new(big.Int).Lsh(big.NewInt(1), 256)       // 2^256
	maxBigInt := new(big.Int).Sub(outOfBounds, big.NewInt(1)) // 2^256 - 1
	maxUint := sdkmath.NewUintFromBigInt(maxBigInt)

	cases := []struct {
		name  string
		value sdkmath.Uint
	}{
		{"zero", sdkmath.ZeroUint()},
		{"one", sdkmath.NewUint(1)},
		{"small", sdkmath.NewUint(100)},
		{"fifty_thousand", sdkmath.NewUint(50_000)},
		{"max", maxUint},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			encoded := UintValueEncoder.Encode(tc.value)
			s.Equal(maxIntKeyLen, len(encoded), "must preserve fixed-width 32-byte format")

			legacy := IntKeyEncoder.Encode(sdkmath.NewIntFromBigInt(tc.value.BigInt()))
			s.Equal(legacy, encoded)

			got := UintValueEncoder.Decode(encoded)
			s.True(tc.value.Equal(got))
			s.Equal(tc.value.String(), UintValueEncoder.Stringify(got))
			s.Equal("math.Uint", UintValueEncoder.Name())
		})
	}

	oldBytesFor100 := make([]byte, maxIntKeyLen)
	oldBytesFor100[maxIntKeyLen-1] = 0x64 // 100
	s.True(sdkmath.NewUint(100).Equal(UintValueEncoder.Decode(oldBytesFor100)))

	s.Panics(func() {
		UintValueEncoder.Encode(sdkmath.Uint{})
	})
	s.Panics(func() {
		UintValueEncoder.Decode([]byte{0x01})
	})
}
