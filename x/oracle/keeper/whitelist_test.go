package keeper

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/set"
)

func TestKeeper_GetVoteTargets(t *testing.T) {
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
			input := CreateTestFixture(t)

			for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[asset.Pair]{}).Keys() {
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
	input := CreateTestFixture(t)

	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[asset.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	validPairs := []asset.Pair{"foo:bar", "xxx:yyy", "whoo:whoo"}
	for _, target := range validPairs {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
		require.True(t, input.OracleKeeper.IsWhitelistedPair(input.Ctx, target))
	}
}

func TestUpdateWhitelist(t *testing.T) {
	fixture := CreateTestFixture(t)
	// prepare test by resetting the genesis pairs
	for _, p := range fixture.OracleKeeper.WhitelistedPairs.Iterate(fixture.Ctx, collections.Range[asset.Pair]{}).Keys() {
		fixture.OracleKeeper.WhitelistedPairs.Delete(fixture.Ctx, p)
	}

	currentWhitelist := set.New(asset.NewPair(denoms.NIBI, denoms.USD), asset.NewPair(denoms.BTC, denoms.USD))
	for p := range currentWhitelist {
		fixture.OracleKeeper.WhitelistedPairs.Insert(fixture.Ctx, p)
	}

	nextWhitelist := set.New(asset.NewPair(denoms.NIBI, denoms.USD), asset.NewPair(denoms.BTC, denoms.USD))

	// no updates case
	whitelistSlice := nextWhitelist.ToSlice()
	sort.Slice(whitelistSlice, func(i, j int) bool {
		return whitelistSlice[i].String() < whitelistSlice[j].String()
	})
	fixture.OracleKeeper.refreshWhitelist(fixture.Ctx, whitelistSlice, currentWhitelist)
	assert.Equal(t, whitelistSlice, fixture.OracleKeeper.GetWhitelistedPairs(fixture.Ctx))

	// len update (fast path)
	nextWhitelist.Add(asset.NewPair(denoms.NIBI, denoms.ETH))
	whitelistSlice = nextWhitelist.ToSlice()
	sort.Slice(whitelistSlice, func(i, j int) bool {
		return whitelistSlice[i].String() < whitelistSlice[j].String()
	})
	fixture.OracleKeeper.refreshWhitelist(fixture.Ctx, whitelistSlice, currentWhitelist)
	assert.Equal(t, whitelistSlice, fixture.OracleKeeper.GetWhitelistedPairs(fixture.Ctx))

	// diff update (slow path)
	currentWhitelist.Add(asset.NewPair(denoms.NIBI, denoms.ATOM))
	whitelistSlice = nextWhitelist.ToSlice()
	sort.Slice(whitelistSlice, func(i, j int) bool {
		return whitelistSlice[i].String() < whitelistSlice[j].String()
	})
	fixture.OracleKeeper.refreshWhitelist(fixture.Ctx, whitelistSlice, currentWhitelist)
	assert.Equal(t, whitelistSlice, fixture.OracleKeeper.GetWhitelistedPairs(fixture.Ctx))
}
