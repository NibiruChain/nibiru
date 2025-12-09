package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSequence(t *testing.T) {
	sk, ctx, _ := deps()
	s := NewSequence(sk, 0)
	// assert initial start number
	require.Equal(t, DefaultSequenceStart, s.Peek(ctx))
	// assert next reports the default sequence start number
	i := s.Next(ctx)
	require.Equal(t, DefaultSequenceStart, i)
	// assert if we peek next number is DefaultSequenceStart + 1
	require.Equal(t, DefaultSequenceStart+1, s.Peek(ctx))
	// assert set correctly does hard reset
	s.Set(ctx, 100)
	require.Equal(t, uint64(100), s.Peek(ctx))
}
