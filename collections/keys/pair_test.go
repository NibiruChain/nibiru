package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPair(t *testing.T) {
	// we only care about bijectivity
	// as Pair is strictly K1, K2 implementation reliant.
	var p = Join(StringKey("hi"), Join(StringKey("hi"), StringKey("hi")))
	bytes := p.KeyBytes()
	idx, result := p.FromKeyBytes(bytes)
	require.Equalf(t, p, result, "%s <-> %s", p.String(), result.String())
	require.Equal(t, len(bytes)-1, idx)
}
