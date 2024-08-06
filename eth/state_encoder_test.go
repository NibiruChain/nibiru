package eth_test

import (
	"testing"

	"github.com/NibiruChain/collections"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/eth"
)

func assertBijectiveKey[T any](t *testing.T, encoder collections.KeyEncoder[T], key T) {
	encodedKey := encoder.Encode(key)
	readLen, decodedKey := encoder.Decode(encodedKey)
	require.Equal(t, len(encodedKey), readLen, "encoded key and read bytes must have same size")
	require.Equal(t, key, decodedKey, "encoding and decoding produces different keys")
	wantStr := encoder.Stringify(key)
	gotStr := encoder.Stringify(decodedKey)
	require.Equal(t, wantStr, gotStr,
		"encoding and decoding produce different string representations")
}

func assertBijectiveValue[T any](t *testing.T, encoder collections.ValueEncoder[T], value T) {
	encodedValue := encoder.Encode(value)
	decodedValue := encoder.Decode(encodedValue)
	require.Equal(t, value, decodedValue, "encoding and decoding produces different values")

	wantStr := encoder.Stringify(value)
	gotStr := encoder.Stringify(decodedValue)
	require.Equal(t, wantStr, gotStr,
		"encoding and decoding produce different string representations")
	require.NotEmpty(t, encoder.Name())
}

func (s *Suite) TestEncoderBytes() {
	testCases := []struct {
		name  string
		value string
	}{
		{"dec-like number", "-1000.5858"},
		{"Nibiru bech32 addr", "nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l"},
		{"Nibiru EVM addr", "0xA52c829E935C30F4C7dcD66739Cf91BF79dD9253"},
		{"normal text with special symbols", "abc123日本語!!??foobar"},
	}
	for _, tc := range testCases {
		s.Run("bijectivity: []byte encoders "+tc.name, func() {
			given := []byte(tc.value)
			assertBijectiveKey(s.T(), eth.KeyEncoderBytes, given)
			assertBijectiveValue(s.T(), eth.ValueEncoderBytes, given)
		})
	}
}

func (s *Suite) TestEncoderEthAddr() {
	testCases := []struct {
		name      string
		given     gethcommon.Address
		wantPanic bool
	}{
		{
			name:  "Nibiru EVM addr",
			given: gethcommon.BytesToAddress([]byte("0xA52c829E935C30F4C7dcD66739Cf91BF79dD9253")),
		},
		{
			name:  "Nibiru EVM addr length above 20 bytes",
			given: gethcommon.BytesToAddress([]byte("0xA52c829E935C30F4C7dcD66739Cf91BF79dD92532456BF123421")),
		},
		{
			name:  "Nibiru Bech 32 addr (hypothetically)",
			given: gethcommon.Address([]byte("nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l")),
		},
	}
	for _, tc := range testCases {
		s.Run("bijectivity: []byte encoders "+tc.name, func() {
			given := tc.given
			runTest := func() {
				assertBijectiveKey(s.T(), eth.KeyEncoderEthAddr, given)
				assertBijectiveValue(s.T(), eth.ValueEncoderEthAddr, given)
			}
			if tc.wantPanic {
				s.Require().Panics(runTest)
			} else {
				s.Require().NotPanics(runTest)
			}
		})
	}
}
