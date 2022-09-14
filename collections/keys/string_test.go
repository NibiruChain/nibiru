package keys

import (
	"bytes"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringKey(t *testing.T) {
	t.Run("bijective", func(t *testing.T) {
		x := StringKey("test")
		i, b := x.FromKeyBytes(x.KeyBytes())
		require.Equal(t, x, b)
		require.Equal(t, 5, i)
	})

	t.Run("panics", func(t *testing.T) {
		// invalid string key
		require.Panics(t, func() {
			invalid := []byte{0x1, 0x0, 0x3}
			StringKey(invalid).KeyBytes()
		})
		// invalid bytes do not end with 0x0
		require.Panics(t, func() {
			StringKey("").FromKeyBytes([]byte{0x1, 0x2})
		})
		// invalid size
		require.Panics(t, func() {
			StringKey("").FromKeyBytes([]byte{1})
		})
	})

	t.Run("proper ordering", func(t *testing.T) {
		stringKeys := []StringKey{
			"a", "aa", "b", "c", "dd",
			"1", "2", "3", "55", StringKey([]byte{1}),
		}

		strings := make([]string, len(stringKeys))
		bytesStringKeys := make([][]byte, len(stringKeys))
		for i, stringKey := range stringKeys {
			strings[i] = string(stringKey)
			bytesStringKeys[i] = stringKey.KeyBytes()
		}

		sort.Strings(strings)
		sort.Slice(bytesStringKeys, func(i, j int) bool {
			return bytes.Compare(bytesStringKeys[i], bytesStringKeys[j]) < 0
		})

		for i, b := range bytesStringKeys {
			expected := strings[i]
			got := string(b[:len(b)-1]) // removes null termination
			require.Equal(t, expected, got)
		}
	})
}
