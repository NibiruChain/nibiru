package collections

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
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

func (s *SuiteValueEncoder) TestUintValueEncoder() {
	maxUint := sdkmath.NewUintFromBigInt(
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), sdkmath.MaxBitLen), big.NewInt(1)),
	)

	testCases := []struct {
		name string
		u    sdkmath.Uint
	}{
		{"zero", sdkmath.ZeroUint()},
		{"one", sdkmath.OneUint()},
		{"small", sdkmath.NewUint(100)},
		{"medium", sdkmath.NewUint(50_000)},
		{"max", maxUint},
	}

	for _, tc := range testCases {
		s.Run("legacy fixed-width round-trip "+tc.name, func() {
			legacyBytes := IntKeyEncoder.Encode(sdkmath.NewIntFromBigInt(tc.u.BigInt()))
			got := UintValueEncoder.Decode(legacyBytes)
			s.Equal(tc.u, got)

			s.Equal(legacyBytes, UintValueEncoder.Encode(tc.u))
			s.Len(legacyBytes, maxIntKeyLen)
		})
	}

	s.Run("bijectivity", func() {
		assertValueBijective(s.T(), UintValueEncoder, sdkmath.NewUint(50_000))
	})

	s.Panics(func() {
		UintValueEncoder.Encode(sdkmath.Uint{})
	})

	s.Equal("math.Uint (fixed-width BE)", UintValueEncoder.Name())
}

func (s *SuiteValueEncoder) TestIntValueEncoder() {
	testCases := []struct {
		name  string
		value sdkmath.Int
	}{
		{"zero", sdkmath.ZeroInt()},
		{"positive", sdkmath.NewInt(50_000)},
		{"negative", sdkmath.NewInt(-1)},
		{"large negative", sdkmath.NewInt(-1_000_000_000_000)},
	}
	for _, tc := range testCases {
		s.Run("signed round-trip "+tc.name, func() {
			assertValueBijective(s.T(), IntValueEncoder, tc.value)
		})
	}

	s.Run("differs from fixed-width uint encoding for positive values", func() {
		v := sdkmath.NewInt(50_000)
		s.NotEqual(IntKeyEncoder.Encode(v), IntValueEncoder.Encode(v))
	})

	s.Equal("math.Int (signed)", IntValueEncoder.Name())
}

func (s *SuiteValueEncoder) TestUintToIntConversion() {
	u := sdkmath.NewUint(12345)
	i := sdkmath.NewIntFromBigInt(u.BigInt())
	s.Equal("12345", i.String())
}
