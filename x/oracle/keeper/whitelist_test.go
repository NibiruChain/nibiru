package keeper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func TestKeeperGetVoteTargets(t *testing.T) {
	type TestCase struct {
		name  string
		in    []asset.Pair
		panic bool
	}

	panicCases := []TestCase{
		{name: "blank pair", in: []asset.Pair{""}, panic: true},
		{name: "blank pair and others", in: []asset.Pair{"", "x", "abc", "defafask"}, panic: true},
		{name: "denom len too short", in: []asset.Pair{"x:y", "xx:yy"}, panic: true},
	}
	happyCases := []TestCase{
		{name: "happy", in: []asset.Pair{"foo:bar", "whoo:whoo"}},
	}

	for _, testCase := range append(panicCases, happyCases...) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			input := CreateTestInput(t)

			for _, p := range input.OracleKeeper.GetWhitelistedPairs(input.Ctx) {
				input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
			}

			expectedTargets := tc.in
			for _, target := range expectedTargets {
				input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
			}

			if tc.panic {
				assert.Panics(t, func() {
					input.OracleKeeper.GetWhitelistedPairs(input.Ctx)
				})
			} else {
				assert.NotPanics(t, func() {
					targets := input.OracleKeeper.GetWhitelistedPairs(input.Ctx)
					assert.Equal(t, expectedTargets, targets)
				})
			}
		})
	}

	input := CreateTestInput(t)

	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[asset.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	expectedTargets := []asset.Pair{"foo:bar", "whoo:whoo"}
	for _, target := range expectedTargets {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
	}

	targets := input.OracleKeeper.GetWhitelistedPairs(input.Ctx)
	require.Equal(t, expectedTargets, targets)
}

func TestIsWhitelistedPair(t *testing.T) {
	input := CreateTestInput(t)

	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[asset.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	validPairs := []asset.Pair{"foo:bar", "xxx:yyy", "whoo:whoo"}
	for _, target := range validPairs {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
		require.True(t, input.OracleKeeper.isWhitelistedPair(input.Ctx, target))
	}
}
