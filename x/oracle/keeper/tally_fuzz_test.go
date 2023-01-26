package keeper

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestFuzz_Tally(t *testing.T) {
	validators := map[string]int64{}

	f := fuzz.New().NilChance(0).Funcs(
		func(e *sdk.Dec, c fuzz.Continue) {
			*e = sdk.NewDec(c.Int63())
		},
		func(e *map[string]int64, c fuzz.Continue) {
			numValidators := c.Intn(100) + 5

			for i := 0; i < numValidators; i++ {
				(*e)[sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()] = c.Int63n(100)
			}
		},
		func(e *map[string]types.ValidatorPerformance, c fuzz.Continue) {
			for validator, power := range validators {
				addr, err := sdk.ValAddressFromBech32(validator)
				require.NoError(t, err)
				(*e)[validator] = types.NewValidatorPerformance(power, addr)
			}
		},
		func(e *types.ExchangeRateBallots, c fuzz.Continue) {
			ballot := types.ExchangeRateBallots{}
			for addr, power := range validators {
				addr, _ := sdk.ValAddressFromBech32(addr)

				var rate sdk.Dec
				c.Fuzz(&rate)

				ballot = append(ballot, types.NewExchangeRateBallot(rate, common.NewAssetPair(c.RandString(), c.RandString()), addr, power))
			}

			*e = ballot
		},
	)

	// set random pairs and validators
	f.Fuzz(&validators)

	claimMap := map[string]types.ValidatorPerformance{}
	f.Fuzz(&claimMap)

	ballot := types.ExchangeRateBallots{}
	f.Fuzz(&ballot)

	var rewardBand sdk.Dec
	f.Fuzz(&rewardBand)

	require.NotPanics(t, func() {
		Tally(ballot, rewardBand, claimMap)
	})
}

func TestOraclePairsInsert(t *testing.T) {
	testCases := []common.AssetPair{"", "1", "22", "2xxxx12312u30912u01u2309u21093u"}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("key: %s", tc), func(t *testing.T) {
			testSetup, _ := setup(t)
			ctx := testSetup.Ctx
			oracleKeeper := testSetup.OracleKeeper

			assert.NotPanics(t, func() {
				oracleKeeper.WhitelistedPairs.Insert(ctx, tc)
			}, "key: %s", tc)
			assert.True(t, oracleKeeper.WhitelistedPairs.Has(ctx, tc))
		})
	}
}

type VoteMap = map[common.AssetPair]types.ExchangeRateBallots

func TestRemoveInvalidBallots(t *testing.T) {
	testCases := []struct {
		name    string
		voteMap VoteMap
	}{
		{
			name: "empty key, empty ballot", voteMap: VoteMap{
				"": types.ExchangeRateBallots{},
			},
		},
		{
			name: "nonempty key, empty ballot", voteMap: VoteMap{
				"xxx": types.ExchangeRateBallots{},
			},
		},
		{
			name: "nonempty keys, empty ballot", voteMap: VoteMap{
				"xxx":    types.ExchangeRateBallots{},
				"abc123": types.ExchangeRateBallots{},
			},
		},
		{
			name: "mixed empty keys, empty ballot", voteMap: VoteMap{
				"xxx":    types.ExchangeRateBallots{},
				"":       types.ExchangeRateBallots{},
				"abc123": types.ExchangeRateBallots{},
				"0x":     types.ExchangeRateBallots{},
			},
		},
		{
			name: "empty key, nonempty ballot, not whitelisted",
			voteMap: VoteMap{
				"": types.ExchangeRateBallots{
					{Pair: "", ExchangeRate: sdk.ZeroDec(), Voter: sdk.ValAddress{}, Power: 0},
				},
			},
		},
		{
			name: "nonempty key, nonempty ballot, whitelisted",
			voteMap: VoteMap{
				"x": types.ExchangeRateBallots{
					{Pair: "x", ExchangeRate: sdk.Dec{}, Voter: sdk.ValAddress{123}, Power: 5},
				},
				common.AssetRegistry.Pair(denoms.DenomBTC, denoms.DenomNUSD): types.ExchangeRateBallots{
					{Pair: common.AssetRegistry.Pair(denoms.DenomBTC, denoms.DenomNUSD), ExchangeRate: sdk.Dec{}, Voter: sdk.ValAddress{123}, Power: 5},
				},
				common.AssetRegistry.Pair(denoms.DenomETH, denoms.DenomNUSD): types.ExchangeRateBallots{
					{Pair: common.AssetRegistry.Pair(denoms.DenomBTC, denoms.DenomNUSD), ExchangeRate: sdk.Dec{}, Voter: sdk.ValAddress{123}, Power: 5},
				},
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			testSetup, _ := setup(t)
			ctx := testSetup.Ctx
			oracleKeeper := testSetup.OracleKeeper

			switch {
			// case tc.err:
			// TODO Include the error case when collections no longer panics
			default:
				assert.NotPanics(t, func() {
					_, _ = oracleKeeper.RemoveInvalidBallots(ctx, tc.voteMap)
				}, "voteMap: %v", tc.voteMap)
			}
		})
	}
}

func TestFuzz_PickReferencePair(t *testing.T) {
	var pairs []common.AssetPair

	f := fuzz.New().NilChance(0).Funcs(
		func(e *common.AssetPair, c fuzz.Continue) {
			*e = common.NewAssetPair(testutil.RandStringBytes(5), testutil.RandStringBytes(5))
		},
		func(e *[]common.AssetPair, c fuzz.Continue) {
			numPairs := c.Intn(100) + 5

			for i := 0; i < numPairs; i++ {
				*e = append(*e, common.NewAssetPair(testutil.RandStringBytes(5), testutil.RandStringBytes(5)))
			}
		},
		func(e *sdk.Dec, c fuzz.Continue) {
			*e = sdk.NewDec(c.Int63())
		},
		func(e *map[common.AssetPair]sdk.Dec, c fuzz.Continue) {
			for _, pair := range pairs {
				var rate sdk.Dec
				c.Fuzz(&rate)

				(*e)[pair] = rate
			}
		},
		func(e *map[string]int64, c fuzz.Continue) {
			numValidator := c.Intn(100) + 5
			for i := 0; i < numValidator; i++ {
				(*e)[sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()] = int64(c.Intn(100) + 1)
			}
		},
		func(e *map[common.AssetPair]types.ExchangeRateBallots, c fuzz.Continue) {
			validators := map[string]int64{}
			c.Fuzz(&validators)

			for _, pair := range pairs {
				ballots := types.ExchangeRateBallots{}

				for addr, power := range validators {
					addr, _ := sdk.ValAddressFromBech32(addr)

					var rate sdk.Dec
					c.Fuzz(&rate)

					ballots = append(ballots, types.NewExchangeRateBallot(rate, pair, addr, power))
				}

				(*e)[pair] = ballots
			}
		},
	)

	// set random pairs
	f.Fuzz(&pairs)

	input, _ := setup(t)

	// test OracleKeeper.Pairs.Insert
	voteTargets := map[common.AssetPair]struct{}{}
	f.Fuzz(&voteTargets)
	whitelistedPairs := make(set.Set[string])

	for key := range voteTargets {
		assert.NotPanics(t, func() {
			input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, key)
		}, "attempted to insert key: %s", key)
		whitelistedPairs.Add(key.String())
	}

	// test OracleKeeper.RemoveInvalidBallots
	voteMap := map[common.AssetPair]types.ExchangeRateBallots{}
	f.Fuzz(&voteMap)

	// Prevent collections error that arrises from iterating over a store with blank keys
	// > Panic value: (blank string here) invalid StringKey bytes. StringKey must be at least length 2.
	var panicAssertFn func(t assert.TestingT, f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
	panicAssertFn = assert.NotPanics
	if whitelistedPairs.Has("") {
		panicAssertFn = assert.Panics
	}
	panicAssertFn(t, func() {
		input.OracleKeeper.RemoveInvalidBallots(input.Ctx, voteMap)
	}, "voteMap: %v", voteMap)
}
