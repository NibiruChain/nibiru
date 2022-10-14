package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRangeBounds(t *testing.T) {
	sk, ctx, _ := deps()

	ks := NewKeySet[uint64](sk, 0, Uint64KeyEncoder)

	ks.Insert(ctx, 1)
	ks.Insert(ctx, 2)
	ks.Insert(ctx, 3)
	ks.Insert(ctx, 4)
	ks.Insert(ctx, 5)
	ks.Insert(ctx, 6)

	// let's range (1-5]; expected: 2..5
	result := ks.Iterate(ctx, Range[uint64]{}.StartExclusive(1).EndInclusive(5)).Keys()
	require.Equal(t, []uint64{2, 3, 4, 5}, result)

	// let's range [1-5); expected 1..4
	result = ks.Iterate(ctx, Range[uint64]{}.StartInclusive(1).EndExclusive(5)).Keys()
	require.Equal(t, []uint64{1, 2, 3, 4}, result)

	// let's range [1-5) descending; expected 4..1
	result = ks.Iterate(ctx, Range[uint64]{}.StartInclusive(1).EndExclusive(5).Descending()).Keys()
	require.Equal(t, []uint64{4, 3, 2, 1}, result)
}
