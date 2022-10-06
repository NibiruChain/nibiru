package keys_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"
)

func TestPairRange(t *testing.T) {
	join := func(a keys.StringKey, b keys.Uint64Key) keys.Pair[keys.StringKey, keys.Uint64Key] {
		return keys.Join(a, b)
	}
	sk, ctx, cdc := deps()

	ks := collections.NewKeySet[keys.Pair[keys.StringKey, keys.Uint64Key]](cdc, sk, 0)
	items := []keys.Pair[keys.StringKey, keys.Uint64Key]{
		join("a", 0),
		join("aa", 1),
		join("aa", 2),
		join("aa", 3),
	}

	for _, i := range items {
		ks.Insert(ctx, i)
	}

	// prefix test
	results := ks.Iterate(ctx, keys.PairRange[keys.StringKey, keys.Uint64Key]{}.Prefix("a")).Keys()
	require.Equal(t, []keys.Pair[keys.StringKey, keys.Uint64Key]{items[0]}, results)

	// start inclusive end inclusive
	rng := keys.PairRange[keys.StringKey, keys.Uint64Key]{}.
		Prefix("aa").
		StartInclusive(1).
		EndInclusive(2)
	results = ks.Iterate(ctx, rng).Keys()
	require.Equal(t, []keys.Pair[keys.StringKey, keys.Uint64Key]{items[1], items[2]}, results)

	// start exclusive end exclusive
	rng = rng.StartExclusive(1).EndExclusive(3).Prefix("aa")
	results = ks.Iterate(ctx, rng).Keys()
	// we expect only 'aa2'
	require.Equal(t, []keys.Pair[keys.StringKey, keys.Uint64Key]{items[2]}, results)

	// reverse
	rng = rng.StartInclusive(0).EndInclusive(3).Descending().Prefix("aa") // note item[0] prefix is 'a' not 'aa'
	results = ks.Iterate(ctx, rng).Keys()
	require.Equal(t, []keys.Pair[keys.StringKey, keys.Uint64Key]{items[3], items[2], items[1]}, results)

	// panics if prefix is not set but any of start or end is
	require.Panics(t, func() {
		rng := keys.PairRange[keys.StringKey, keys.Uint64Key]{}.StartInclusive(0)
		rng.RangeValues()
	})
	require.Panics(t, func() {
		rng := keys.PairRange[keys.StringKey, keys.Uint64Key]{}.StartExclusive(0)
		rng.RangeValues()
	})
	require.Panics(t, func() {
		rng := keys.PairRange[keys.StringKey, keys.Uint64Key]{}.EndInclusive(0)
		rng.RangeValues()
	})
	require.Panics(t, func() {
		rng := keys.PairRange[keys.StringKey, keys.Uint64Key]{}.EndExclusive(0)
		rng.RangeValues()
	})
}
