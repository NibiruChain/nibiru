package keeper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"
)

func TestKeeper_GetVoteTargets(t *testing.T) {
	type TestCase struct {
		name  string
		in    []string
		panic bool
	}

	panicCases := []TestCase{
		{name: "blank pair", in: []string{""}, panic: true},
		{name: "blank pair and others", in: []string{"", "x", "abc", "defafask"}, panic: true},
	}
	happyCases := []TestCase{
		{name: "happy", in: []string{"bar", "foo", "whoowhoo"}},
		{name: "short len 1 pair", in: []string{"x"}},
		{name: "short len 2 pair", in: []string{"xx"}},
	}

	for _, testCase := range append(panicCases, happyCases...) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			input := CreateTestInput(t)

			for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[string]{}).Keys() {
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

	input := CreateTestInput(t)

	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[string]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	expectedTargets := []string{"bar", "foo", "whoowhoo"}
	for _, target := range expectedTargets {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
	}

	targets := input.OracleKeeper.GetWhitelistedPairs(input.Ctx)
	require.Equal(t, expectedTargets, targets)
}

func TestKeeper_IsVoteTarget(t *testing.T) {
	input := CreateTestInput(t)

	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[string]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	validTargets := []string{"bar", "foo", "whoowhoo"}
	for _, target := range validTargets {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
		require.True(t, input.OracleKeeper.IsWhitelistedPair(input.Ctx, target))
	}
}
