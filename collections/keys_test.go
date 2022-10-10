package collections

import (
	"bytes"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint64(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		key := uint64(0x0123456789ABCDEF)
		bytes := uint64Key{}.KeyEncode(key)
		idx, result := uint64Key{}.KeyDecode(bytes)
		require.Equalf(t, key, result, "%d <-> %d", key, result)
		require.Equal(t, 8, idx)
	})

	t.Run("empty", func(t *testing.T) {
		var k uint64
		require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0}, uint64Key{}.KeyEncode(k))
	})
}

func TestStringKey(t *testing.T) {
	t.Run("bijective", func(t *testing.T) {
		x := "test"
		i, b := stringKey{}.KeyDecode(stringKey{}.KeyEncode(x))
		require.Equal(t, x, b)
		require.Equal(t, 5, i)
	})

	t.Run("panics", func(t *testing.T) {
		// invalid string key
		require.Panics(t, func() {
			invalid := []byte{0x1, 0x0, 0x3}
			stringKey{}.KeyEncode(string(invalid))
		})
		// invalid bytes do not end with 0x0
		require.Panics(t, func() {
			stringKey{}.KeyDecode([]byte{0x1, 0x2})
		})
		// invalid size
		require.Panics(t, func() {
			stringKey{}.KeyDecode([]byte{0x1})
		})
	})

	t.Run("proper ordering", func(t *testing.T) {
		stringKeys := []string{
			"a", "aa", "b", "c", "dd",
			"1", "2", "3", "55", string([]byte{1}),
		}

		strings := make([]string, len(stringKeys))
		bytesStringKeys := make([][]byte, len(stringKeys))
		for i, sk := range stringKeys {
			strings[i] = sk
			bytesStringKeys[i] = stringKey{}.KeyEncode(sk)
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
