package keeper

import (
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
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

func TestGroupBallotsByPair(t *testing.T) {
	fixture := CreateTestFixture(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	sh := stakingkeeper.NewMsgServerImpl(fixture.StakingKeeper)

	// Validator created
	_, err := sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	staking.EndBlocker(fixture.Ctx, fixture.StakingKeeper)

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

	for i, ballot := range btcBallots {
		fixture.OracleKeeper.Votes.Insert(
			fixture.Ctx,
			ValAddrs[i],
			types.NewAggregateExchangeRateVote(
				types.ExchangeRateTuples{
					{Pair: ballot.Pair, ExchangeRate: ballot.ExchangeRate},
					{Pair: ethBallots[i].Pair, ExchangeRate: ethBallots[i].ExchangeRate},
				},
				ValAddrs[i],
			),
		)
	}

	// organize votes by pair
	ballotMap := fixture.OracleKeeper.groupBallotsByPair(fixture.Ctx, types.ValidatorPerformances{
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
	fixture := CreateTestFixture(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	sh := stakingkeeper.NewMsgServerImpl(fixture.StakingKeeper)

	// Validator created
	_, err := sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	staking.EndBlocker(fixture.Ctx, fixture.StakingKeeper)

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
		fixture.OracleKeeper.Prevotes.Insert(fixture.Ctx, ValAddrs[i], types.AggregateExchangeRatePrevote{
			Hash:        "",
			Voter:       ValAddrs[i].String(),
			SubmitBlock: uint64(fixture.Ctx.BlockHeight()),
		})

		fixture.OracleKeeper.Votes.Insert(fixture.Ctx, ValAddrs[i],
			types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{
				{Pair: btcBallot[i].Pair, ExchangeRate: btcBallot[i].ExchangeRate},
				{Pair: ethBallot[i].Pair, ExchangeRate: ethBallot[i].ExchangeRate},
			}, ValAddrs[i]))
	}

	fixture.OracleKeeper.clearVotesAndPreVotes(fixture.Ctx, 10)

	prevoteCounter := len(fixture.OracleKeeper.Prevotes.Iterate(fixture.Ctx, collections.Range[sdk.ValAddress]{}).Keys())
	voteCounter := len(fixture.OracleKeeper.Votes.Iterate(fixture.Ctx, collections.Range[sdk.ValAddress]{}).Keys())

	require.Equal(t, prevoteCounter, 3)
	require.Equal(t, voteCounter, 0)

	// vote period starts at b=10, clear the votes at b=0 and below.
	fixture.OracleKeeper.clearVotesAndPreVotes(fixture.Ctx.WithBlockHeight(fixture.Ctx.BlockHeight()+10), 10)
	prevoteCounter = len(fixture.OracleKeeper.Prevotes.Iterate(fixture.Ctx, collections.Range[sdk.ValAddress]{}).Keys())
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

type VoteMap = map[asset.Pair]types.ExchangeRateBallots

func TestRemoveInvalidBallots(t *testing.T) {
	testCases := []struct {
		name    string
		voteMap VoteMap
	}{
		{
			name: "empty key, empty ballot",
			voteMap: VoteMap{
				"": types.ExchangeRateBallots{},
			},
		},
		{
			name: "nonempty key, empty ballot",
			voteMap: VoteMap{
				"xxx": types.ExchangeRateBallots{},
			},
		},
		{
			name: "nonempty keys, empty ballot",
			voteMap: VoteMap{
				"xxx":    types.ExchangeRateBallots{},
				"abc123": types.ExchangeRateBallots{},
			},
		},
		{
			name: "mixed empty keys, empty ballot",
			voteMap: VoteMap{
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
			fixture, _ := Setup(t)
			assert.NotPanics(t, func() {
				fixture.OracleKeeper.removeInvalidBallots(fixture.Ctx, tc.voteMap)
			}, "voteMap: %v", tc.voteMap)
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
			for i := 0; i < 5+c.Intn(100); i++ {
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
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, key)
		whitelistedPairs.Add(key.String())
	}

	// test OracleKeeper.RemoveInvalidBallots
	voteMap := map[asset.Pair]types.ExchangeRateBallots{}
	f.Fuzz(&voteMap)

	assert.NotPanics(t, func() {
		input.OracleKeeper.removeInvalidBallots(input.Ctx, voteMap)
	}, "voteMap: %v", voteMap)
}

func TestZeroBallotPower(t *testing.T) {
	btcBallots := types.ExchangeRateBallots{
		types.NewExchangeRateBallot(sdk.NewDec(17), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[0], 0),
		types.NewExchangeRateBallot(sdk.NewDec(10), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[1], 0),
		types.NewExchangeRateBallot(sdk.NewDec(6), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[2], 0),
	}

	assert.False(t, isPassingVoteThreshold(btcBallots, sdk.ZeroInt(), 0))
}
