package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPair(t *testing.T) {
	// we only care about bijectivity
	// as Pair is strictly K1, K2 implementation reliant.

	t.Run("joined", func(t *testing.T) {
		p := Join(StringKey("hi"), Join(StringKey("hi"), StringKey("hi")))
		bytes := p.KeyBytes()
		idx, result := p.FromKeyBytes(bytes)
		require.Equalf(t, p, result, "%s <-> %s", p.String(), result.String())
		require.Equal(t, len(bytes), idx)
	})

	t.Run("pair prefix", func(t *testing.T) {
		k1 := StringKey("hi")
		prefix := PairPrefix[StringKey, Uint64Key](k1)
		require.Equal(t, k1.KeyBytes(), prefix.KeyBytes())
	})

	t.Run("pair suffix", func(t *testing.T) {
		k2 := Uint64Key(10)
		suffix := PairSuffix[StringKey, Uint64Key](k2)
		require.Equal(t, k2.KeyBytes(), suffix.KeyBytes())
	})

	t.Run("empty", func(t *testing.T) {
		var p Pair[StringKey, StringKey]
		require.Panics(t, func() {
			p.KeyBytes()
		})
	})
}
