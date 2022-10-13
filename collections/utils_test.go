package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func assertBijective[T any](t *testing.T, encoder KeyEncoder[T], key T) {
	encodedKey := encoder.Encode(key)
	read, decodedKey := encoder.Decode(encodedKey)
	require.Equal(t, len(encodedKey), read, "encoded key and read bytes must have same size")
	require.Equal(t, key, decodedKey, "encoding and decoding produces different keys")
}

func assertValueBijective[T any](t *testing.T, encoder ValueEncoder[T], value T) {
	encodedValue := encoder.ValueEncode(value)
	decodedValue := encoder.ValueDecode(encodedValue)
	require.Equal(t, value, decodedValue, "encoding and decoding produces different values")
}
