package keeper

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/cometbft/cometbft/libs/rand"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

func TestOracleThreshold(t *testing.T) {
	exchangeRates := types.ExchangeRateTuples{
		{
			Pair:         asset.Registry.Pair(denoms.BTC, denoms.USD),
			ExchangeRate: testExchangeRate,
		},
	}
	exchangeRateStr, err := exchangeRates.ToString()
	require.NoError(t, err)

	fixture, msgServer := Setup(t)
	params, _ := fixture.OracleKeeper.Params.Get(fixture.Ctx)
	params.ExpirationBlocks = 0
	fixture.OracleKeeper.Params.Set(fixture.Ctx, params)

	// Case 1.
	// Less than the threshold signs, exchange rate consensus fails
	for i := 0; i < 1; i++ {
		salt := fmt.Sprintf("%d", i)
		hash := types.GetAggregateVoteHash(salt, exchangeRateStr, ValAddrs[i])
		prevoteMsg := types.NewMsgAggregateExchangeRatePrevote(hash, Addrs[i], ValAddrs[i])
		voteMsg := types.NewMsgAggregateExchangeRateVote(salt, exchangeRateStr, Addrs[i], ValAddrs[i])

		_, err1 := msgServer.AggregateExchangeRatePrevote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(0)), prevoteMsg)
		_, err2 := msgServer.AggregateExchangeRateVote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(1)), voteMsg)
		require.NoError(t, err1)
		require.NoError(t, err2)
	}
	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)
	_, err = fixture.OracleKeeper.ExchangeRates.Get(fixture.Ctx.WithBlockHeight(1), exchangeRates[0].Pair)
	assert.Error(t, err)

	// Case 2.
	// More than the threshold signs, exchange rate consensus succeeds
	for i := 0; i < 4; i++ {
		salt := fmt.Sprintf("%d", i)
		hash := types.GetAggregateVoteHash(salt, exchangeRateStr, ValAddrs[i])
		prevoteMsg := types.NewMsgAggregateExchangeRatePrevote(hash, Addrs[i], ValAddrs[i])
		voteMsg := types.NewMsgAggregateExchangeRateVote(salt, exchangeRateStr, Addrs[i], ValAddrs[i])

		_, err1 := msgServer.AggregateExchangeRatePrevote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(0)), prevoteMsg)
		_, err2 := msgServer.AggregateExchangeRateVote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(1)), voteMsg)
		require.NoError(t, err1)
		require.NoError(t, err2)
	}
	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)
	rate, err := fixture.OracleKeeper.ExchangeRates.Get(fixture.Ctx, exchangeRates[0].Pair)
	require.NoError(t, err)
	assert.Equal(t, testExchangeRate, rate.ExchangeRate)

	// Case 3.
	// Increase voting power of absent validator, exchange rate consensus fails
	val, _ := fixture.StakingKeeper.GetValidator(fixture.Ctx, ValAddrs[4])
	_, _ = fixture.StakingKeeper.Delegate(fixture.Ctx.WithBlockHeight(0), Addrs[4], testStakingAmt.MulRaw(8), stakingtypes.Unbonded, val, false)

	for i := 0; i < 4; i++ {
		salt := fmt.Sprintf("%d", i)
		hash := types.GetAggregateVoteHash(salt, exchangeRateStr, ValAddrs[i])
		prevoteMsg := types.NewMsgAggregateExchangeRatePrevote(hash, Addrs[i], ValAddrs[i])
		voteMsg := types.NewMsgAggregateExchangeRateVote(salt, exchangeRateStr, Addrs[i], ValAddrs[i])

		_, err1 := msgServer.AggregateExchangeRatePrevote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(0)), prevoteMsg)
		_, err2 := msgServer.AggregateExchangeRateVote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(1)), voteMsg)
		require.NoError(t, err1)
		require.NoError(t, err2)
	}
	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)
	_, err = fixture.OracleKeeper.ExchangeRates.Get(fixture.Ctx, exchangeRates[0].Pair)
	assert.Error(t, err)
}

func TestResetExchangeRates(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.USD)
	fixture, _ := Setup(t)

	emptyVotes := map[asset.Pair]types.ExchangeRateVotes{}
	validVotes := map[asset.Pair]types.ExchangeRateVotes{pair: {}}

	// Set expiration blocks to 10
	params, _ := fixture.OracleKeeper.Params.Get(fixture.Ctx)
	params.ExpirationBlocks = 10
	fixture.OracleKeeper.Params.Set(fixture.Ctx, params)

	// Post a price at block 1
	fixture.OracleKeeper.SetPrice(fixture.Ctx.WithBlockHeight(1), pair, testExchangeRate)

	// reset exchange rates at block 2
	// Price should still be there because not expired yet
	fixture.OracleKeeper.clearExchangeRates(fixture.Ctx.WithBlockHeight(2), emptyVotes)
	_, err := fixture.OracleKeeper.ExchangeRates.Get(fixture.Ctx, pair)
	assert.NoError(t, err)

	// reset exchange rates at block 3 but pair is in votes
	// Price should be removed there because there was a valid votes
	fixture.OracleKeeper.clearExchangeRates(fixture.Ctx.WithBlockHeight(3), validVotes)
	_, err = fixture.OracleKeeper.ExchangeRates.Get(fixture.Ctx, pair)
	assert.Error(t, err)

	// Post a price at block 69
	// reset exchange rates at block 79
	// Price should not be there anymore because expired
	fixture.OracleKeeper.SetPrice(fixture.Ctx.WithBlockHeight(69), pair, testExchangeRate)
	fixture.OracleKeeper.clearExchangeRates(fixture.Ctx.WithBlockHeight(79), emptyVotes)

	_, err = fixture.OracleKeeper.ExchangeRates.Get(fixture.Ctx, pair)
	assert.Error(t, err)
}

func TestOracleTally(t *testing.T) {
	fixture, _ := Setup(t)

	votes := types.ExchangeRateVotes{}
	rates, valAddrs, stakingKeeper := types.GenerateRandomTestCase()
	fixture.OracleKeeper.StakingKeeper = stakingKeeper
	h := NewMsgServerImpl(fixture.OracleKeeper)

	for i, rate := range rates {
		decExchangeRate := sdkmath.LegacyNewDecWithPrec(int64(rate*math.Pow10(OracleDecPrecision)), int64(OracleDecPrecision))
		exchangeRateStr, err := types.ExchangeRateTuples{
			{ExchangeRate: decExchangeRate, Pair: asset.Registry.Pair(denoms.BTC, denoms.USD)},
		}.ToString()
		require.NoError(t, err)

		salt := fmt.Sprintf("%d", i)
		hash := types.GetAggregateVoteHash(salt, exchangeRateStr, valAddrs[i])
		prevoteMsg := types.NewMsgAggregateExchangeRatePrevote(hash, sdk.AccAddress(valAddrs[i]), valAddrs[i])
		voteMsg := types.NewMsgAggregateExchangeRateVote(salt, exchangeRateStr, sdk.AccAddress(valAddrs[i]), valAddrs[i])

		_, err1 := h.AggregateExchangeRatePrevote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(0)), prevoteMsg)
		_, err2 := h.AggregateExchangeRateVote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(1)), voteMsg)
		require.NoError(t, err1)
		require.NoError(t, err2)

		power := testStakingAmt.QuoRaw(int64(6)).Int64()
		if decExchangeRate.IsZero() {
			power = int64(0)
		}

		vote := types.NewExchangeRateVote(
			decExchangeRate, asset.Registry.Pair(denoms.BTC, denoms.USD), valAddrs[i], power)
		votes = append(votes, vote)

		// change power of every three validator
		if i%3 == 0 {
			stakingKeeper.Validators()[i].SetConsensusPower(int64(i + 1))
		}
	}

	validatorPerformances := make(types.ValidatorPerformances)
	for _, valAddr := range valAddrs {
		validatorPerformances[valAddr.String()] = types.NewValidatorPerformance(
			stakingKeeper.Validator(fixture.Ctx, valAddr).GetConsensusPower(sdk.DefaultPowerReduction),
			valAddr,
		)
	}
	sort.Sort(votes)
	weightedMedian := votes.WeightedMedianWithAssertion()
	standardDeviation := votes.StandardDeviation(weightedMedian)
	maxSpread := weightedMedian.Mul(fixture.OracleKeeper.RewardBand(fixture.Ctx).QuoInt64(2))

	if standardDeviation.GT(maxSpread) {
		maxSpread = standardDeviation
	}

	expectedValidatorPerformances := make(types.ValidatorPerformances)
	for _, valAddr := range valAddrs {
		expectedValidatorPerformances[valAddr.String()] = types.NewValidatorPerformance(
			stakingKeeper.Validator(fixture.Ctx, valAddr).GetConsensusPower(sdk.DefaultPowerReduction),
			valAddr,
		)
	}

	for _, vote := range votes {
		key := vote.Voter.String()
		validatorPerformance := expectedValidatorPerformances[key]
		if vote.ExchangeRate.GTE(weightedMedian.Sub(maxSpread)) &&
			vote.ExchangeRate.LTE(weightedMedian.Add(maxSpread)) {
			validatorPerformance.RewardWeight += vote.Power
			validatorPerformance.WinCount++
		} else if !vote.ExchangeRate.IsPositive() {
			validatorPerformance.AbstainCount++
		} else {
			validatorPerformance.MissCount++
		}
		expectedValidatorPerformances[key] = validatorPerformance
	}

	tallyMedian := Tally(
		votes, fixture.OracleKeeper.RewardBand(fixture.Ctx), validatorPerformances)

	assert.Equal(t, expectedValidatorPerformances, validatorPerformances)
	assert.Equal(t, tallyMedian.MulInt64(100).TruncateInt(), weightedMedian.MulInt64(100).TruncateInt())
	assert.NotEqualValues(t, 0, validatorPerformances.TotalRewardWeight(), validatorPerformances.String())
}

func TestOracleRewardBand(t *testing.T) {
	fixture, msgServer := Setup(t)
	params, err := fixture.OracleKeeper.Params.Get(fixture.Ctx)
	require.NoError(t, err)

	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.ATOM, denoms.USD)}
	fixture.OracleKeeper.Params.Set(fixture.Ctx, params)

	// clear pairs to reset vote targets
	for _, p := range fixture.OracleKeeper.WhitelistedPairs.Iterate(fixture.Ctx, collections.Range[asset.Pair]{}).Keys() {
		fixture.OracleKeeper.WhitelistedPairs.Delete(fixture.Ctx, p)
	}
	fixture.OracleKeeper.WhitelistedPairs.Insert(fixture.Ctx, asset.Registry.Pair(denoms.ATOM, denoms.USD))

	rewardSpread := testExchangeRate.Mul(fixture.OracleKeeper.RewardBand(fixture.Ctx).QuoInt64(2))

	// Account 1, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate.Sub(rewardSpread)},
	}, 0)

	// Account 2, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
	}, 1)

	// Account 3, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
	}, 2)

	// Account 4, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate.Add(rewardSpread)},
	}, 3)

	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[0], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[1], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[3], 0))

	// Account 1 will miss the vote due to raward band condition
	// Account 1, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate.Sub(rewardSpread.Add(sdkmath.LegacyOneDec()))},
	}, 0)

	// Account 2, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
	}, 1)

	// Account 3, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
	}, 2)

	// Account 4, atom:usd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate.Add(rewardSpread)},
	}, 3)

	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[0], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[1], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[3], 0))
}

/* TODO(Mercilex): not appliable right now: https://github.com/NibiruChain/nibiru/issues/805
func TestOracleMultiRewardDistribution(t *testing.T) {
	input, h := setup(t)

	// SDR and KRW have the same voting power, but KRW has been chosen as referencepair by alphabetical order.
	// Account 1, SDR, KRW
	makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: common.Pairbtc:usd.String(), ExchangeRate: randomExchangeRate}, {Pair: common.Pairatom:usd.String(), ExchangeRate: randomExchangeRate}}, 0)

	// Account 2, SDR
	makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: common.Pairbtc:usd.String(), ExchangeRate: randomExchangeRate}}, 1)

	// Account 3, KRW
	makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: common.Pairbtc:usd.String(), ExchangeRate: randomExchangeRate}}, 2)

	rewardAmt := math.NewInt(1e6)
	err := input.BankKeeper.MintCoins(input.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(denoms.Gov, rewardAmt)))
	require.NoError(t, err)

	input.OracleKeeper.UpdateExchangeRates(input.Ctx)

	rewardDistributedWindow := input.OracleKeeper.RewardDistributionWindow(input.Ctx)

	expectedRewardAmt := math.LegacyNewDecFromInt(rewardAmt.QuoRaw(3).MulRaw(2)).QuoInt64(int64(rewardDistributedWindow)).TruncateInt()
	expectedRewardAmt2 := math.ZeroInt() // even vote power is same KRW with SDR, KRW chosen referenceTerra because alphabetical order
	expectedRewardAmt3 := math.LegacyNewDecFromInt(rewardAmt.QuoRaw(3)).QuoInt64(int64(rewardDistributedWindow)).TruncateInt()

	rewards := input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[0])
	assert.Equal(t, expectedRewardAmt, rewards.Rewards.AmountOf(denoms.Gov).TruncateInt())
	rewards = input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[1])
	assert.Equal(t, expectedRewardAmt2, rewards.Rewards.AmountOf(denoms.Gov).TruncateInt())
	rewards = input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[2])
	assert.Equal(t, expectedRewardAmt3, rewards.Rewards.AmountOf(denoms.Gov).TruncateInt())
}
*/

func TestOracleExchangeRate(t *testing.T) {
	// The following scenario tests four validators providing prices for eth:usd, atom:usd, and btc:usd.
	// eth:usd and atom:usd pass, but btc:usd fails due to not enough validators voting.
	input, h := Setup(t)

	atomUsdExchangeRate := sdkmath.LegacyNewDec(1000000)
	ethUsdExchangeRate := sdkmath.LegacyNewDec(1000000)
	btcusdExchangeRate := sdkmath.LegacyNewDec(1e6)

	// Account 1, eth:usd, atom:usd, btc:usd
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.USD), ExchangeRate: ethUsdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: atomUsdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.USD), ExchangeRate: btcusdExchangeRate},
	}, 0)

	// Account 2, eth:usd, atom:usd
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.USD), ExchangeRate: ethUsdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: atomUsdExchangeRate},
	}, 1)

	// Account 3, eth:usd, atom:usd, btc:usd(abstain)
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.USD), ExchangeRate: ethUsdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: atomUsdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.USD), ExchangeRate: sdkmath.LegacyZeroDec()},
	}, 2)

	// Account 4, eth:usd, atom:usd, btc:usd
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.USD), ExchangeRate: ethUsdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: atomUsdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.USD), ExchangeRate: sdkmath.LegacyZeroDec()},
	}, 3)

	ethUsdRewards := sdk.NewInt64Coin("ETHREWARD", 1*common.TO_MICRO)
	atomUsdRewards := sdk.NewInt64Coin("ATOMREWARD", 1*common.TO_MICRO)

	AllocateRewards(t, input, sdk.NewCoins(ethUsdRewards), 1)
	AllocateRewards(t, input, sdk.NewCoins(atomUsdRewards), 1)

	input.OracleKeeper.UpdateExchangeRates(input.Ctx)

	// total reward pool for the current vote period is 1* common.TO_MICRO for eth:usd and 1* common.TO_MICRO for atom:usd
	// val 1,2,3,4 all won on 2 pairs
	// so total votes are 2 * 2 + 2 + 2 = 8
	expectedRewardAmt := sdk.NewDecCoinsFromCoins(ethUsdRewards, atomUsdRewards).
		QuoDec(sdkmath.LegacyNewDec(8)). // total votes
		MulDec(sdkmath.LegacyNewDec(2))  // votes won by val1 and val2
	rewards := input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[0])
	assert.Equalf(t, expectedRewardAmt, rewards.Rewards, "%s <-> %s", expectedRewardAmt, rewards.Rewards)
	rewards = input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[1])
	assert.Equalf(t, expectedRewardAmt, rewards.Rewards, "%s <-> %s", expectedRewardAmt, rewards.Rewards)
	rewards = input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[2])
	assert.Equalf(t, expectedRewardAmt, rewards.Rewards, "%s <-> %s", expectedRewardAmt, rewards.Rewards)
	rewards = input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[3])
	assert.Equalf(t, expectedRewardAmt, rewards.Rewards, "%s <-> %s", expectedRewardAmt, rewards.Rewards)
}

func TestOracleRandomPrices(t *testing.T) {
	fixture, msgServer := Setup(t)

	for i := 0; i < 100; i++ {
		for val := 0; val < 4; val++ {
			MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
				{Pair: asset.Registry.Pair(denoms.ETH, denoms.USD), ExchangeRate: sdkmath.LegacyNewDec(int64(rand.Uint64() % 1e6))},
				{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: sdkmath.LegacyNewDec(int64(rand.Uint64() % 1e6))},
			}, val)
		}

		require.NotPanics(t, func() {
			fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)
		})
	}
}

func TestWhitelistedPairs(t *testing.T) {
	fixture, msgServer := Setup(t)
	params, err := fixture.OracleKeeper.Params.Get(fixture.Ctx)
	require.NoError(t, err)

	t.Log("whitelist ONLY atom:usd")
	for _, p := range fixture.OracleKeeper.WhitelistedPairs.Iterate(fixture.Ctx, collections.Range[asset.Pair]{}).Keys() {
		fixture.OracleKeeper.WhitelistedPairs.Delete(fixture.Ctx, p)
	}
	fixture.OracleKeeper.WhitelistedPairs.Insert(fixture.Ctx, asset.Registry.Pair(denoms.ATOM, denoms.USD))

	t.Log("vote and prevote from all vals on atom:usd")
	priceVoteFromVal := func(valIdx int, block int64) {
		MakeAggregatePrevoteAndVote(t, fixture, msgServer, block, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate}}, valIdx)
	}
	block := int64(0)
	priceVoteFromVal(0, block)
	priceVoteFromVal(1, block)
	priceVoteFromVal(2, block)
	priceVoteFromVal(3, block)

	t.Log("whitelist btc:usd for next vote period")
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.ATOM, denoms.USD), asset.Registry.Pair(denoms.BTC, denoms.USD)}
	fixture.OracleKeeper.Params.Set(fixture.Ctx, params)
	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	t.Log("assert: no miss counts for all vals")
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[0], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[1], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[3], 0))

	t.Log("whitelisted pairs are {atom:usd, btc:usd}")
	assert.Equal(t,
		[]asset.Pair{
			asset.Registry.Pair(denoms.ATOM, denoms.USD),
			asset.Registry.Pair(denoms.BTC, denoms.USD),
		},
		fixture.OracleKeeper.GetWhitelistedPairs(fixture.Ctx))

	t.Log("vote from vals 0-3 on atom:usd (but not btc:usd)")
	priceVoteFromVal(0, block)
	priceVoteFromVal(1, block)
	priceVoteFromVal(2, block)
	priceVoteFromVal(3, block)

	t.Log("delete btc:usd for next vote period")
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.ATOM, denoms.USD)}
	fixture.OracleKeeper.Params.Set(fixture.Ctx, params)
	perfs := fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	t.Log("validators 0-3 all voted -> expect win")
	for valIdx := 0; valIdx < 4; valIdx++ {
		perf := perfs[ValAddrs[valIdx].String()]
		assert.EqualValues(t, 1, perf.WinCount)
		assert.EqualValues(t, 1, perf.AbstainCount)
		assert.EqualValues(t, 0, perf.MissCount)
	}
	t.Log("validators 4 didn't vote -> expect abstain")
	perf := perfs[ValAddrs[4].String()]
	assert.EqualValues(t, 0, perf.WinCount)
	assert.EqualValues(t, 2, perf.AbstainCount)
	assert.EqualValues(t, 0, perf.MissCount)

	t.Log("btc:usd must be deleted")
	assert.Equal(t, []asset.Pair{asset.Registry.Pair(denoms.ATOM, denoms.USD)},
		fixture.OracleKeeper.GetWhitelistedPairs(fixture.Ctx))
	require.False(t, fixture.OracleKeeper.WhitelistedPairs.Has(
		fixture.Ctx, asset.Registry.Pair(denoms.BTC, denoms.USD)))

	t.Log("vote from vals 0-3 on atom:usd")
	priceVoteFromVal(0, block)
	priceVoteFromVal(1, block)
	priceVoteFromVal(2, block)
	priceVoteFromVal(3, block)
	perfs = fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	t.Log("Although validators 0-2 voted, it's for the same period -> expect abstains for everyone")
	for valIdx := 0; valIdx < 4; valIdx++ {
		perf := perfs[ValAddrs[valIdx].String()]
		assert.EqualValues(t, 1, perf.WinCount)
		assert.EqualValues(t, 0, perf.AbstainCount)
		assert.EqualValues(t, 0, perf.MissCount)
	}
	perf = perfs[ValAddrs[4].String()]
	assert.EqualValues(t, 0, perf.WinCount)
	assert.EqualValues(t, 1, perf.AbstainCount)
	assert.EqualValues(t, 0, perf.MissCount)
}
