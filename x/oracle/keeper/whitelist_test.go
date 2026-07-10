package keeper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/collections"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

func TestKeeper_GetVoteTargets(t *testing.T) {
	type TestCase struct {
		name  string
		in    []types.Pair
		panic bool
	}

	panicCases := []TestCase{
		{name: "blank pair", in: []types.Pair{""}, panic: true},
		{name: "blank pair and others", in: []types.Pair{"", "x", "abc", "defafask"}, panic: true},
		{name: "denom len too short", in: []types.Pair{"x:y", "xx:yy"}, panic: true},
	}
	happyCases := []TestCase{
		{name: "happy", in: []types.Pair{"foo:bar", "whoo:whoo"}},
	}

	for _, testCase := range append(panicCases, happyCases...) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			input := CreateTestFixture(t)

			for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[types.Pair]{}).Keys() {
				input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
			}

			expectedTargets := tc.in
			for _, target := range expectedTargets {
				input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
			}

			var panicAssertFn func(t assert.TestingT, f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
			switch tc.panic {
			case true:
				panicAssertFn = assert.Panics
			default:
				panicAssertFn = assert.NotPanics
			}
			panicAssertFn(t, func() {
				targets := input.OracleKeeper.GetWhitelistedPairs(input.Ctx)
				assert.Equal(t, expectedTargets, targets)
			})
		})
	}

	input := CreateTestFixture(t)

	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[types.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	expectedTargets := []types.Pair{"foo:bar", "whoo:whoo"}
	for _, target := range expectedTargets {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
	}

	targets := input.OracleKeeper.GetWhitelistedPairs(input.Ctx)
	require.Equal(t, expectedTargets, targets)
}

func TestIsWhitelistedPair(t *testing.T) {
	input := CreateTestFixture(t)

	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[types.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	validPairs := []types.Pair{"foo:bar", "xxx:yyy", "whoo:whoo"}
	for _, target := range validPairs {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
		require.True(t, input.OracleKeeper.IsWhitelistedPair(input.Ctx, target))
	}
}
