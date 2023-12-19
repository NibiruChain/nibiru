package keeper

import (
	"sort"
	"testing"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

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

func TestGroupVotesByPair(t *testing.T) {
	fixture := CreateTestFixture(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	sh := stakingkeeper.NewMsgServerImpl(&fixture.StakingKeeper)

	// Validator created
	_, err := sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	staking.EndBlocker(fixture.Ctx, &fixture.StakingKeeper)

	pairBtc := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	pairEth := asset.Registry.Pair(denoms.ETH, denoms.NUSD)
	btcVotes := types.ExchangeRateVotes{
		{Pair: pairBtc, ExchangeRate: sdk.NewDec(17), Voter: ValAddrs[0], Power: power},
		{Pair: pairBtc, ExchangeRate: sdk.NewDec(10), Voter: ValAddrs[1], Power: power},
		{Pair: pairBtc, ExchangeRate: sdk.NewDec(6), Voter: ValAddrs[2], Power: power},
	}
	ethVotes := types.ExchangeRateVotes{
		{Pair: pairEth, ExchangeRate: sdk.NewDec(1_000), Voter: ValAddrs[0], Power: power},
		{Pair: pairEth, ExchangeRate: sdk.NewDec(1_300), Voter: ValAddrs[1], Power: power},
		{Pair: pairEth, ExchangeRate: sdk.NewDec(2_000), Voter: ValAddrs[2], Power: power},
	}

	for i, v := range btcVotes {
		fixture.OracleKeeper.Votes.Insert(
			fixture.Ctx,
			ValAddrs[i],
			types.NewAggregateExchangeRateVote(
				types.ExchangeRateTuples{
					{Pair: v.Pair, ExchangeRate: v.ExchangeRate},
					{Pair: ethVotes[i].Pair, ExchangeRate: ethVotes[i].ExchangeRate},
				},
				ValAddrs[i],
			),
		)
	}

	// organize votes by pair
	pairVotes := fixture.OracleKeeper.groupVotesByPair(fixture.Ctx, types.ValidatorPerformances{
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

	// sort each votes for comparison
	sort.Sort(btcVotes)
	sort.Sort(ethVotes)
	sort.Sort(pairVotes[asset.Registry.Pair(denoms.BTC, denoms.NUSD)])
	sort.Sort(pairVotes[asset.Registry.Pair(denoms.ETH, denoms.NUSD)])

	require.Equal(t, btcVotes, pairVotes[asset.Registry.Pair(denoms.BTC, denoms.NUSD)])
	require.Equal(t, ethVotes, pairVotes[asset.Registry.Pair(denoms.ETH, denoms.NUSD)])
}

func TestClearVotesAndPrevotes(t *testing.T) {
	fixture := CreateTestFixture(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	sh := stakingkeeper.NewMsgServerImpl(&fixture.StakingKeeper)

	// Validator created
	_, err := sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	staking.EndBlocker(fixture.Ctx, &fixture.StakingKeeper)

	btcVotes := types.ExchangeRateVotes{
		types.NewExchangeRateVote(sdk.NewDec(17), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[0], power),
		types.NewExchangeRateVote(sdk.NewDec(10), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[1], power),
		types.NewExchangeRateVote(sdk.NewDec(6), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[2], power),
	}
	ethVotes := types.ExchangeRateVotes{
		types.NewExchangeRateVote(sdk.NewDec(1000), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[0], power),
		types.NewExchangeRateVote(sdk.NewDec(1300), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[1], power),
		types.NewExchangeRateVote(sdk.NewDec(2000), asset.Registry.Pair(denoms.ETH, denoms.NUSD), ValAddrs[2], power),
	}

	for i := range btcVotes {
		fixture.OracleKeeper.Prevotes.Insert(fixture.Ctx, ValAddrs[i], types.AggregateExchangeRatePrevote{
			Hash:        "",
			Voter:       ValAddrs[i].String(),
			SubmitBlock: uint64(fixture.Ctx.BlockHeight()),
		})

		fixture.OracleKeeper.Votes.Insert(fixture.Ctx, ValAddrs[i],
			types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{
				{Pair: btcVotes[i].Pair, ExchangeRate: btcVotes[i].ExchangeRate},
				{Pair: ethVotes[i].Pair, ExchangeRate: ethVotes[i].ExchangeRate},
			}, ValAddrs[i]))
	}

	fixture.OracleKeeper.clearVotesAndPrevotes(fixture.Ctx, 10)

	prevoteCounter := len(fixture.OracleKeeper.Prevotes.Iterate(fixture.Ctx, collections.Range[sdk.ValAddress]{}).Keys())
	voteCounter := len(fixture.OracleKeeper.Votes.Iterate(fixture.Ctx, collections.Range[sdk.ValAddress]{}).Keys())

	require.Equal(t, prevoteCounter, 3)
	require.Equal(t, voteCounter, 0)

	// vote period starts at b=10, clear the votes at b=0 and below.
	fixture.OracleKeeper.clearVotesAndPrevotes(fixture.Ctx.WithBlockHeight(fixture.Ctx.BlockHeight()+10), 10)
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
		func(e *types.ExchangeRateVotes, c fuzz.Continue) {
			votes := types.ExchangeRateVotes{}
			for addr, power := range validators {
				addr, _ := sdk.ValAddressFromBech32(addr)

				var rate sdk.Dec
				c.Fuzz(&rate)

				votes = append(votes, types.NewExchangeRateVote(rate, asset.NewPair(c.RandString(), c.RandString()), addr, power))
			}

			*e = votes
		},
	)

	// set random pairs and validators
	f.Fuzz(&validators)

	claimMap := types.ValidatorPerformances{}
	f.Fuzz(&claimMap)

	votes := types.ExchangeRateVotes{}
	f.Fuzz(&votes)

	var rewardBand sdk.Dec
	f.Fuzz(&rewardBand)

	require.NotPanics(t, func() {
		Tally(votes, rewardBand, claimMap)
	})
}

type VoteMap = map[asset.Pair]types.ExchangeRateVotes

func TestRemoveInvalidBallots(t *testing.T) {
	testCases := []struct {
		name    string
		voteMap VoteMap
	}{
		{
			name: "empty key, empty votes",
			voteMap: VoteMap{
				"": types.ExchangeRateVotes{},
			},
		},
		{
			name: "nonempty key, empty votes",
			voteMap: VoteMap{
				"xxx": types.ExchangeRateVotes{},
			},
		},
		{
			name: "nonempty keys, empty votes",
			voteMap: VoteMap{
				"xxx":    types.ExchangeRateVotes{},
				"abc123": types.ExchangeRateVotes{},
			},
		},
		{
			name: "mixed empty keys, empty votes",
			voteMap: VoteMap{
				"xxx":    types.ExchangeRateVotes{},
				"":       types.ExchangeRateVotes{},
				"abc123": types.ExchangeRateVotes{},
				"0x":     types.ExchangeRateVotes{},
			},
		},
		{
			name: "empty key, nonempty votes, not whitelisted",
			voteMap: VoteMap{
				"": types.ExchangeRateVotes{
					{Pair: "", ExchangeRate: sdk.ZeroDec(), Voter: sdk.ValAddress{}, Power: 0},
				},
			},
		},
		{
			name: "nonempty key, nonempty votes, whitelisted",
			voteMap: VoteMap{
				"x": types.ExchangeRateVotes{
					{Pair: "x", ExchangeRate: sdk.Dec{}, Voter: sdk.ValAddress{123}, Power: 5},
				},
				asset.Registry.Pair(denoms.BTC, denoms.NUSD): types.ExchangeRateVotes{
					{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.Dec{}, Voter: sdk.ValAddress{123}, Power: 5},
				},
				asset.Registry.Pair(denoms.ETH, denoms.NUSD): types.ExchangeRateVotes{
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
				fixture.OracleKeeper.removeInvalidVotes(fixture.Ctx, tc.voteMap, set.New[asset.Pair](
					asset.NewPair(denoms.BTC, denoms.NUSD),
					asset.NewPair(denoms.ETH, denoms.NUSD),
				))
			}, "voteMap: %v", tc.voteMap)
		})
	}
}

func TestFuzzPickReferencePair(t *testing.T) {
	var pairs []asset.Pair

	f := fuzz.New().NilChance(0).Funcs(
		func(e *asset.Pair, c fuzz.Continue) {
			*e = asset.NewPair(testutil.RandLetters(5), testutil.RandLetters(5))
		},
		func(e *[]asset.Pair, c fuzz.Continue) {
			numPairs := c.Intn(100) + 5

			for i := 0; i < numPairs; i++ {
				*e = append(*e, asset.NewPair(testutil.RandLetters(5), testutil.RandLetters(5)))
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
		func(e *map[asset.Pair]types.ExchangeRateVotes, c fuzz.Continue) {
			validators := map[string]int64{}
			c.Fuzz(&validators)

			for _, pair := range pairs {
				votes := types.ExchangeRateVotes{}

				for addr, power := range validators {
					addr, _ := sdk.ValAddressFromBech32(addr)

					var rate sdk.Dec
					c.Fuzz(&rate)

					votes = append(votes, types.NewExchangeRateVote(rate, pair, addr, power))
				}

				(*e)[pair] = votes
			}
		},
	)

	// set random pairs
	f.Fuzz(&pairs)

	input, _ := Setup(t)

	// test OracleKeeper.Pairs.Insert
	voteTargets := set.Set[asset.Pair]{}
	f.Fuzz(&voteTargets)
	whitelistedPairs := make(set.Set[asset.Pair])

	for key := range voteTargets {
		whitelistedPairs.Add(key)
	}

	// test OracleKeeper.RemoveInvalidBallots
	voteMap := map[asset.Pair]types.ExchangeRateVotes{}
	f.Fuzz(&voteMap)

	assert.NotPanics(t, func() {
		input.OracleKeeper.removeInvalidVotes(input.Ctx, voteMap, whitelistedPairs)
	}, "voteMap: %v", voteMap)
}

func TestZeroBallotPower(t *testing.T) {
	btcVotess := types.ExchangeRateVotes{
		types.NewExchangeRateVote(sdk.NewDec(17), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[0], 0),
		types.NewExchangeRateVote(sdk.NewDec(10), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[1], 0),
		types.NewExchangeRateVote(sdk.NewDec(6), asset.Registry.Pair(denoms.BTC, denoms.NUSD), ValAddrs[2], 0),
	}

	assert.False(t, isPassingVoteThreshold(btcVotess, sdk.ZeroInt(), 0))
}
