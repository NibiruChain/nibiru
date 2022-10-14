package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPairKeyEncoder(t *testing.T) {
	// we only care about bijectivity
	// as Pair is strictly K1, K2 implementation reliant.

	enc := PairKeyEncoder[string, string](StringKeyEncoder, StringKeyEncoder)

	t.Run("encode both - bijectivity", func(t *testing.T) {
		key := Join("k1", "k2")
		b := enc.Encode(key)
		read, got := enc.Decode(b)
		require.Equal(t, key, got)
		require.Equal(t, len(b), read)
	})

	t.Run("encode partial - k1", func(t *testing.T) {
		key := PairPrefix[string, string]("k1")
		require.Equal(t, StringKeyEncoder.Encode("k1"), enc.Encode(key))
	})

	t.Run("encode partial - k2", func(t *testing.T) {
		key := PairSuffix[string, string]("k2")
		require.Equal(t, StringKeyEncoder.Encode("k2"), enc.Encode(key))
	})

	t.Run("empty panics", func(t *testing.T) {
		require.Panics(t, func() {
			enc.Encode(Pair[string, string]{})
		})
	})

	t.Run("stringify both", func(t *testing.T) {
		s := enc.Stringify(Join("k1", "k2"))
		require.Equal(t, `("k1", "k2")`, s)
	})

	t.Run("stringify k1", func(t *testing.T) {
		s := enc.Stringify(PairPrefix[string, string]("k1"))
		require.Equal(t, `("k1", <nil>)`, s)
	})
	t.Run("stringify k2", func(t *testing.T) {
		s := enc.Stringify(PairSuffix[string, string]("k2"))
		require.Equal(t, `(<nil>, "k2")`, s)
	})
}

func TestPairRange(t *testing.T) {
	sk, ctx, _ := deps()

	ks := NewKeySet[Pair[string, uint64]](
		sk,
		0,
		PairKeyEncoder[string, uint64](StringKeyEncoder, Uint64KeyEncoder),
	)
	items := []Pair[string, uint64]{
		Join("a", uint64(0)),
		Join("aa", uint64(1)),
		Join("aa", uint64(2)),
		Join("aa", uint64(3)),
	}

	for _, i := range items {
		ks.Insert(ctx, i)
	}

	// prefix test
	results := ks.Iterate(ctx, PairRange[string, uint64]{}.Prefix("a")).Keys()
	require.Equal(t, []Pair[string, uint64]{items[0]}, results)

	// start inclusive end inclusive
	rng := PairRange[string, uint64]{}.
		Prefix("aa").
		StartInclusive(1).
		EndInclusive(2)
	results = ks.Iterate(ctx, rng).Keys()
	require.Equal(t, []Pair[string, uint64]{items[1], items[2]}, results)

	// start exclusive end exclusive
	rng = rng.StartExclusive(1).EndExclusive(3).Prefix("aa")
	results = ks.Iterate(ctx, rng).Keys()
	// we expect only 'aa2'
	require.Equal(t, []Pair[string, uint64]{items[2]}, results)

	// reverse
	rng = rng.StartInclusive(0).EndInclusive(3).Descending().Prefix("aa") // note item[0] prefix is 'a' not 'aa'
	results = ks.Iterate(ctx, rng).Keys()
	require.Equal(t, []Pair[string, uint64]{items[3], items[2], items[1]}, results)

	// panics if prefix is not set but any of start or end is
	require.Panics(t, func() {
		rng := PairRange[string, uint64]{}.StartInclusive(0)
		rng.RangeValues()
	})
	require.Panics(t, func() {
		rng := PairRange[string, uint64]{}.StartExclusive(0)
		rng.RangeValues()
	})
	require.Panics(t, func() {
		rng := PairRange[string, uint64]{}.EndInclusive(0)
		rng.RangeValues()
	})
	require.Panics(t, func() {
		rng := PairRange[string, uint64]{}.EndExclusive(0)
		rng.RangeValues()
	})
}
