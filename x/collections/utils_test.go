package collections

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func assertBijective[T any](t *testing.T, encoder KeyEncoder[T], key T) {
	encodedKey, err := encoder.Encode(key)
	require.NoError(t, err)
	read, decodedKey := encoder.Decode(encodedKey)
	require.Equal(t, len(encodedKey), read, "encoded key and read bytes must have same size")
	require.Equal(t, key, decodedKey, "encoding and decoding produces different keys")
}

func assertValueBijective[T any](t *testing.T, encoder ValueEncoder[T], value T) {
	encodedValue, err := encoder.Encode(value)
	require.NoError(t, err)
	decodedValue := encoder.Decode(encodedValue)
	require.Equal(t, value, decodedValue, "encoding and decoding produces different values")
}

// stringValue is a ValueEncoder for string, used for testing.
type stringValue struct{}

func (s stringValue) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (s stringValue) Decode(b []byte) string              { return string(b) }
func (s stringValue) Stringify(value string) string       { return value }
func (s stringValue) Name() string                        { return "test string" }

// jsonValue is a ValueEncoder for objects to be turned into json.
// used for testing.
type jsonValue[T any] struct{}

func (jsonValue[T]) Encode(value T) ([]byte, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (jsonValue[T]) Decode(b []byte) T {
	v := new(T)
	_ = json.Unmarshal(b, v)
	return *v
}

func (jsonValue[T]) Stringify(v T) string { return fmt.Sprintf("%#v", v) }
func (jsonValue[T]) Name() string {
	var t T
	return fmt.Sprintf("json-value-%T", t)
}
