package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPair(t *testing.T) {
	type pairNested = Pair[StringKey, StringKey]
	type pair = Pair[StringKey, pairNested]
	// we only care about bijectivity
	// as this is strictly implementation reliant.
	p := pair{
		P1: String("data"),
		P2: pairNested{
			P1: String("name"),
			P2: String("surname"),
		},
	}
	bytes := p.KeyBytes()
	idx, result := p.FromKeyBytes(bytes)
	require.Equal(t, p, result)
	require.Equal(t, len(bytes)-1, idx)
}
