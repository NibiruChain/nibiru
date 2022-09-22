package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint8(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		key := Uint8Key(0xFF)
		bytes := key.KeyBytes()
		idx, result := key.FromKeyBytes(bytes)
		require.Equalf(t, key, result, "%s <-> %s", key.String(), result.String())
		require.Equal(t, 1, idx)
	})

	t.Run("empty", func(t *testing.T) {
		var k Uint8Key
		require.Equal(t, []byte{0}, k.KeyBytes())
	})
}

func TestUint64(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		key := Uint64Key(0x0123456789ABCDEF)
		bytes := key.KeyBytes()
		idx, result := key.FromKeyBytes(bytes)
		require.Equalf(t, key, result, "%s <-> %s", key.String(), result.String())
		require.Equal(t, 8, idx)
	})

	t.Run("empty", func(t *testing.T) {
		var k Uint64Key
		require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0}, k.KeyBytes())
	})
}
