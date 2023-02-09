package keeper

import (
	"fmt"
	"sort"
	"testing"

	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestOrganizeAggregate(t *testing.T) {
	input := CreateTestFixture(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	sh := staking.NewHandler(input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := sh(ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = sh(ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = sh(ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	staking.EndBlocker(ctx, input.StakingKeeper)

	btcBallots := types.ExchangeRateBallots{
		types.NewExchangeRateBallot(sdk.NewDec(17), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(10), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(6), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[2], power),
	}
	ethBallots := types.ExchangeRateBallots{
		types.NewExchangeRateBallot(sdk.NewDec(1000), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(1300), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(2000), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[2], power),
	}

	for i := range btcBallots {
		input.OracleKeeper.Votes.Insert(
			input.Ctx,
			ValAddrs[i],
			types.NewAggregateExchangeRateVote(
				types.ExchangeRateTuples{
					{Pair: btcBallots[i].Pair, ExchangeRate: btcBallots[i].ExchangeRate},
					{Pair: ethBallots[i].Pair, ExchangeRate: ethBallots[i].ExchangeRate},
				},
				ValAddrs[i],
			),
		)
	}

	// organize votes by pair
	ballotMap := input.OracleKeeper.groupBallotsByPair(input.Ctx, types.ValidatorPerformances{
		ValAddrs[0].String(): {
			Power:      power,
			WinCount:   0,
			ValAddress: ValAddrs[0],
		},
		ValAddrs[1].String(): {
			Power:      power,
			WinCount:   0,
			ValAddress: ValAddrs[1],
		},
		ValAddrs[2].String(): {
			Power:      power,
			WinCount:   0,
			ValAddress: ValAddrs[2],
		},
	})

	// sort each ballot for comparison
	sort.Sort(btcBallots)
	sort.Sort(ethBallots)
	sort.Sort(ballotMap[asset.Registry.Pair(denoms.BTC, denoms.NUSD)])
	sort.Sort(ballotMap[asset.Registry.Pair(denoms.ETH, denoms.NUSD)])

	require.Equal(t, btcBallots, ballotMap[asset.Registry.Pair(denoms.BTC, denoms.NUSD)])
	require.Equal(t, ethBallots, ballotMap[asset.Registry.Pair(denoms.ETH, denoms.NUSD)])
}

func TestClearBallots(t *testing.T) {
	input := CreateTestFixture(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	sh := staking.NewHandler(input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := sh(ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = sh(ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = sh(ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	staking.EndBlocker(ctx, input.StakingKeeper)

	btcBallot := types.ExchangeRateBallots{
		types.NewExchangeRateBallot(sdk.NewDec(17), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(10), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(6), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[2], power),
	}
	ethBallot := types.ExchangeRateBallots{
		types.NewExchangeRateBallot(sdk.NewDec(1000), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(1300), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(2000), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[2], power),
	}

	for i := range btcBallot {
		input.OracleKeeper.Prevotes.Insert(input.Ctx, ValAddrs[i], types.AggregateExchangeRatePrevote{
			Hash:        "",
			Voter:       ValAddrs[i].String(),
			SubmitBlock: uint64(input.Ctx.BlockHeight()),
		})

		input.OracleKeeper.Votes.Insert(input.Ctx, ValAddrs[i],
			types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{
				{Pair: btcBallot[i].Pair, ExchangeRate: btcBallot[i].ExchangeRate},
				{Pair: ethBallot[i].Pair, ExchangeRate: ethBallot[i].ExchangeRate},
			}, ValAddrs[i]))
	}

	input.OracleKeeper.clearVotesAndPreVotes(input.Ctx, 5)

	prevoteCounter := len(input.OracleKeeper.Prevotes.Iterate(input.Ctx, collections.Range[sdk.ValAddress]{}).Keys())
	voteCounter := len(input.OracleKeeper.Votes.Iterate(input.Ctx, collections.Range[sdk.ValAddress]{}).Keys())

	require.Equal(t, prevoteCounter, 3)
	require.Equal(t, voteCounter, 0)

	input.OracleKeeper.clearVotesAndPreVotes(input.Ctx.WithBlockHeight(input.Ctx.BlockHeight()+6), 5)
	prevoteCounter = len(input.OracleKeeper.Prevotes.Iterate(input.Ctx, collections.Range[sdk.ValAddress]{}).Keys())
	require.Equal(t, prevoteCounter, 0)
}

func TestFuzzTally(t *testing.T) {
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
		func(e *types.ValidatorPerformances, c fuzz.Continue) {
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

				ballot = append(ballot, types.NewExchangeRateBallot(rate, asset.NewPair(c.RandString(), c.RandString()), addr, power))
			}

			*e = ballot
		},
	)

	// set random pairs and validators
	f.Fuzz(&validators)

	claimMap := types.ValidatorPerformances{}
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
	testCases := []asset.Pair{"", "1", "22", "2xxxx12312u30912u01u2309u21093u"}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("key: %s", tc), func(t *testing.T) {
			testSetup, _ := Setup(t)
			ctx := testSetup.Ctx
			oracleKeeper := testSetup.OracleKeeper

			assert.NotPanics(t, func() {
				oracleKeeper.WhitelistedPairs.Insert(ctx, tc)
			}, "key: %s", tc)
			assert.True(t, oracleKeeper.WhitelistedPairs.Has(ctx, tc))
		})
	}
}

type VoteMap = map[asset.Pair]types.ExchangeRateBallots

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
				asset.Registry.Pair(denoms.BTC, denoms.NUSD): types.ExchangeRateBallots{
					{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.Dec{}, Voter: sdk.ValAddress{123}, Power: 5},
				},
				asset.Registry.Pair(denoms.ETH, denoms.NUSD): types.ExchangeRateBallots{
					{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.Dec{}, Voter: sdk.ValAddress{123}, Power: 5},
				},
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			testSetup, _ := Setup(t)
			ctx := testSetup.Ctx
			oracleKeeper := testSetup.OracleKeeper

			switch {
			// case tc.err:
			// TODO Include the error case when collections no longer panics
			default:
				assert.NotPanics(t, func() {
					_, _ = oracleKeeper.removeInvalidBallots(ctx, tc.voteMap)
				}, "voteMap: %v", tc.voteMap)
			}
		})
	}
}

func TestFuzzPickReferencePair(t *testing.T) {
	var pairs []asset.Pair

	f := fuzz.New().NilChance(0).Funcs(
		func(e *asset.Pair, c fuzz.Continue) {
			*e = asset.NewPair(testutil.RandStringBytes(5), testutil.RandStringBytes(5))
		},
		func(e *[]asset.Pair, c fuzz.Continue) {
			numPairs := c.Intn(100) + 5

			for i := 0; i < numPairs; i++ {
				*e = append(*e, asset.NewPair(testutil.RandStringBytes(5), testutil.RandStringBytes(5)))
			}
		},
		func(e *sdk.Dec, c fuzz.Continue) {
			*e = sdk.NewDec(c.Int63())
		},
		func(e *map[asset.Pair]sdk.Dec, c fuzz.Continue) {
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
		func(e *map[asset.Pair]types.ExchangeRateBallots, c fuzz.Continue) {
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

	input, _ := Setup(t)

	// test OracleKeeper.Pairs.Insert
	voteTargets := set.Set[asset.Pair]{}
	f.Fuzz(&voteTargets)
	whitelistedPairs := make(set.Set[string])

	for key := range voteTargets {
		assert.NotPanics(t, func() {
			input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, key)
		}, "attempted to insert key: %s", key)
		whitelistedPairs.Add(key.String())
	}

	// test OracleKeeper.RemoveInvalidBallots
	voteMap := map[asset.Pair]types.ExchangeRateBallots{}
	f.Fuzz(&voteMap)

	// Prevent collections error that arrises from iterating over a store with blank keys
	// > Panic value: (blank string here) invalid StringKey bytes. StringKey must be at least length 2.
	var panicAssertFn func(t assert.TestingT, f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
	panicAssertFn = assert.NotPanics
	if whitelistedPairs.Has("") {
		panicAssertFn = assert.Panics
	}
	panicAssertFn(t, func() {
		input.OracleKeeper.removeInvalidBallots(input.Ctx, voteMap)
	}, "voteMap: %v", voteMap)
}
