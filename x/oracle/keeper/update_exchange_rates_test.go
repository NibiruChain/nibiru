package keeper

import (
	"fmt"
	"math"
	"sort"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/rand"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestOracleThreshold(t *testing.T) {
	exchangeRates := types.ExchangeRateTuples{
		{
			Pair:         asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			ExchangeRate: randomExchangeRate,
		},
	}
	exchangeRateStr, err := exchangeRates.ToString()
	require.NoError(t, err)

	fixture, msgServer := Setup(t)

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
	assert.Equal(t, randomExchangeRate, rate)

	// Case 3.
	// Increase voting power of absent validator, exchange rate consensus fails
	val, _ := fixture.StakingKeeper.GetValidator(fixture.Ctx, ValAddrs[4])
	_, _ = fixture.StakingKeeper.Delegate(fixture.Ctx.WithBlockHeight(0), Addrs[4], stakingAmt.MulRaw(8), stakingtypes.Unbonded, val, false)

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

func TestOracleTally(t *testing.T) {
	fixture, _ := Setup(t)

	ballot := types.ExchangeRateBallots{}
	rates, valAddrs, stakingKeeper := types.GenerateRandomTestCase()
	fixture.OracleKeeper.StakingKeeper = stakingKeeper
	h := NewMsgServerImpl(fixture.OracleKeeper)

	for i, rate := range rates {
		decExchangeRate := sdk.NewDecWithPrec(int64(rate*math.Pow10(OracleDecPrecision)), int64(OracleDecPrecision))
		exchangeRateStr, err := types.ExchangeRateTuples{
			{ExchangeRate: decExchangeRate, Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}}.ToString()
		require.NoError(t, err)

		salt := fmt.Sprintf("%d", i)
		hash := types.GetAggregateVoteHash(salt, exchangeRateStr, valAddrs[i])
		prevoteMsg := types.NewMsgAggregateExchangeRatePrevote(hash, sdk.AccAddress(valAddrs[i]), valAddrs[i])
		voteMsg := types.NewMsgAggregateExchangeRateVote(salt, exchangeRateStr, sdk.AccAddress(valAddrs[i]), valAddrs[i])

		_, err1 := h.AggregateExchangeRatePrevote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(0)), prevoteMsg)
		_, err2 := h.AggregateExchangeRateVote(sdk.WrapSDKContext(fixture.Ctx.WithBlockHeight(1)), voteMsg)
		require.NoError(t, err1)
		require.NoError(t, err2)

		power := stakingAmt.QuoRaw(int64(6)).Int64()
		if decExchangeRate.IsZero() {
			power = int64(0)
		}

		vote := types.NewExchangeRateBallot(
			decExchangeRate, asset.Registry.Pair(denoms.BTC, denoms.NUSD), valAddrs[i], power)
		ballot = append(ballot, vote)

		// change power of every three validator
		if i%3 == 0 {
			stakingKeeper.Validators()[i].SetConsensusPower(int64(i + 1))
		}
	}

	validatorClaimMap := make(types.ValidatorPerformances)
	for _, valAddr := range valAddrs {
		validatorClaimMap[valAddr.String()] = types.ValidatorPerformance{
			Power:        stakingKeeper.Validator(fixture.Ctx, valAddr).GetConsensusPower(sdk.DefaultPowerReduction),
			RewardWeight: int64(0),
			WinCount:     int64(0),
			ValAddress:   valAddr,
		}
	}
	sort.Sort(ballot)
	weightedMedian := ballot.WeightedMedianWithAssertion()
	standardDeviation := ballot.StandardDeviation(weightedMedian)
	maxSpread := weightedMedian.Mul(fixture.OracleKeeper.RewardBand(fixture.Ctx).QuoInt64(2))

	if standardDeviation.GT(maxSpread) {
		maxSpread = standardDeviation
	}

	expectedValidatorClaimMap := make(types.ValidatorPerformances)
	for _, valAddr := range valAddrs {
		expectedValidatorClaimMap[valAddr.String()] = types.ValidatorPerformance{
			Power:        stakingKeeper.Validator(fixture.Ctx, valAddr).GetConsensusPower(sdk.DefaultPowerReduction),
			RewardWeight: int64(0),
			WinCount:     int64(0),
			ValAddress:   valAddr,
		}
	}

	for _, vote := range ballot {
		if (vote.ExchangeRate.GTE(weightedMedian.Sub(maxSpread)) &&
			vote.ExchangeRate.LTE(weightedMedian.Add(maxSpread))) ||
			!vote.ExchangeRate.IsPositive() {
			key := vote.Voter.String()
			claim := expectedValidatorClaimMap[key]
			claim.RewardWeight += vote.Power
			claim.WinCount++
			expectedValidatorClaimMap[key] = claim
		}
	}

	tallyMedian := Tally(ballot, fixture.OracleKeeper.RewardBand(fixture.Ctx), validatorClaimMap)

	assert.Equal(t, expectedValidatorClaimMap, validatorClaimMap)
	assert.Equal(t, tallyMedian.MulInt64(100).TruncateInt(), weightedMedian.MulInt64(100).TruncateInt())
}

func TestOracleRewardBand(t *testing.T) {
	fixture, msgServer := Setup(t)
	params := fixture.OracleKeeper.GetParams(fixture.Ctx)
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.NIBI, denoms.NUSD)}
	fixture.OracleKeeper.SetParams(fixture.Ctx, params)

	// clear pairs to reset vote targets
	for _, p := range fixture.OracleKeeper.WhitelistedPairs.Iterate(fixture.Ctx, collections.Range[asset.Pair]{}).Keys() {
		fixture.OracleKeeper.WhitelistedPairs.Delete(fixture.Ctx, p)
	}
	fixture.OracleKeeper.WhitelistedPairs.Insert(fixture.Ctx, asset.Registry.Pair(denoms.NIBI, denoms.NUSD))

	rewardSpread := randomExchangeRate.Mul(fixture.OracleKeeper.RewardBand(fixture.Ctx).QuoInt64(2))

	// Account 1, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate.Sub(rewardSpread)},
	}, 0)

	// Account 2, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate},
	}, 1)

	// Account 3, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate},
	}, 2)

	// Account 4, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate.Add(rewardSpread)},
	}, 3)

	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[0], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[1], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[3], 0))

	// Account 1 will miss the vote due to raward band condition
	// Account 1, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate.Sub(rewardSpread.Add(sdk.OneDec()))},
	}, 0)

	// Account 2, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate},
	}, 1)

	// Account 3, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate},
	}, 2)

	// Account 4, nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate.Add(rewardSpread)},
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
	makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: common.Pairbtc:nusd.String(), ExchangeRate: randomExchangeRate}, {Pair: common.Pairnibi:nusd.String(), ExchangeRate: randomExchangeRate}}, 0)

	// Account 2, SDR
	makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: common.Pairbtc:nusd.String(), ExchangeRate: randomExchangeRate}}, 1)

	// Account 3, KRW
	makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: common.Pairbtc:nusd.String(), ExchangeRate: randomExchangeRate}}, 2)

	rewardAmt := sdk.NewInt(1e6)
	err := input.BankKeeper.MintCoins(input.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(denoms.Gov, rewardAmt)))
	require.NoError(t, err)

	input.OracleKeeper.UpdateExchangeRates(input.Ctx)

	rewardDistributedWindow := input.OracleKeeper.RewardDistributionWindow(input.Ctx)

	expectedRewardAmt := sdk.NewDecFromInt(rewardAmt.QuoRaw(3).MulRaw(2)).QuoInt64(int64(rewardDistributedWindow)).TruncateInt()
	expectedRewardAmt2 := sdk.ZeroInt() // even vote power is same KRW with SDR, KRW chosen referenceTerra because alphabetical order
	expectedRewardAmt3 := sdk.NewDecFromInt(rewardAmt.QuoRaw(3)).QuoInt64(int64(rewardDistributedWindow)).TruncateInt()

	rewards := input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[0])
	assert.Equal(t, expectedRewardAmt, rewards.Rewards.AmountOf(denoms.Gov).TruncateInt())
	rewards = input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[1])
	assert.Equal(t, expectedRewardAmt2, rewards.Rewards.AmountOf(denoms.Gov).TruncateInt())
	rewards = input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(2), ValAddrs[2])
	assert.Equal(t, expectedRewardAmt3, rewards.Rewards.AmountOf(denoms.Gov).TruncateInt())
}
*/

func TestOracleExchangeRate(t *testing.T) {
	// The following scenario tests four validators providing prices for eth:nusd, nibi:nusd, and btc:nusd.
	// eth:nusd and nibi:nusd pass, but btc:nusd fails due to not enough validators voting.
	input, h := Setup(t)

	nibiNusdExchangeRate := sdk.NewDec(1000000)
	ethNusdExchangeRate := sdk.NewDec(1000000)
	btcNusdExchangeRate := sdk.NewDec(1e6)

	// Account 1, eth:nusd, nibi:nusd, btc:nusd
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: ethNusdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: nibiNusdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: btcNusdExchangeRate},
	}, 0)

	// Account 2, eth:nusd, nibi:nusd
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: ethNusdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: nibiNusdExchangeRate},
	}, 1)

	// Account 3, eth:nusd, nibi:nusd, btc:nusd(abstain)
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: ethNusdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: nibiNusdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.ZeroDec()},
	}, 2)

	// Account 4, eth:nusd, nibi:nusd, btc:nusd
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: ethNusdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: nibiNusdExchangeRate},
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.ZeroDec()},
	}, 3)

	ethNusdRewards := sdk.NewInt64Coin("ETHREWARD", 1*common.Precision)
	nibiNusdRewards := sdk.NewInt64Coin("NIBIREWARD", 1*common.Precision)

	AllocateRewards(t, input, asset.Registry.Pair(denoms.ETH, denoms.NUSD), sdk.NewCoins(ethNusdRewards), 1)
	AllocateRewards(t, input, asset.Registry.Pair(denoms.NIBI, denoms.NUSD), sdk.NewCoins(nibiNusdRewards), 1)

	input.OracleKeeper.UpdateExchangeRates(input.Ctx)

	// total reward pool for the current vote period is 1* common.Precision for eth:nusd and 1* common.Precision for nibi:nusd
	// val 1,2,3,4 all won on 2 pairs
	// so total votes are 2 * 2 + 2 + 2 = 8
	expectedRewardAmt := sdk.NewDecCoinsFromCoins(ethNusdRewards, nibiNusdRewards).
		QuoDec(sdk.NewDec(8)). // total votes
		MulDec(sdk.NewDec(2))  // votes won by val1 and val2
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
				{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: sdk.NewDec(int64(rand.Uint64() % 1e6))},
				{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: sdk.NewDec(int64(rand.Uint64() % 1e6))},
			}, val)
		}

		require.NotPanics(t, func() {
			fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)
		})
	}
}

func TestWhitelistedPairs(t *testing.T) {
	fixture, msgServer := Setup(t)
	params := fixture.OracleKeeper.GetParams(fixture.Ctx)

	for _, p := range fixture.OracleKeeper.WhitelistedPairs.Iterate(fixture.Ctx, collections.Range[asset.Pair]{}).Keys() {
		fixture.OracleKeeper.WhitelistedPairs.Delete(fixture.Ctx, p)
	}
	fixture.OracleKeeper.WhitelistedPairs.Insert(fixture.Ctx, asset.Registry.Pair(denoms.NIBI, denoms.NUSD))

	// nibi:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 0)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 1)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 3)

	// add btc:nusd for next vote period
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.NIBI, denoms.NUSD), asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
	fixture.OracleKeeper.SetParams(fixture.Ctx, params)
	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	// no missing current
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[0], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[1], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
	assert.Equal(t, uint64(0), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[3], 0))

	// whitelisted pairs are {nibi:nusd, btc:nusd}
	assert.Equal(t, []asset.Pair{asset.Registry.Pair(denoms.BTC, denoms.NUSD), asset.Registry.Pair(denoms.NIBI, denoms.NUSD)}, fixture.OracleKeeper.GetWhitelistedPairs(fixture.Ctx))

	// nibi:nusd, missing btc:nusd
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 0)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 1)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 3)

	// delete btc:nusd for next vote period
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.NIBI, denoms.NUSD)}
	fixture.OracleKeeper.SetParams(fixture.Ctx, params)
	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[0], 0))
	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[1], 0))
	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[3], 0))

	// btc:nusd must be deleted
	assert.Equal(t, []asset.Pair{asset.Registry.Pair(denoms.NIBI, denoms.NUSD)}, fixture.OracleKeeper.GetWhitelistedPairs(fixture.Ctx))
	require.False(t, fixture.OracleKeeper.WhitelistedPairs.Has(fixture.Ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)))

	// nibi:nusd, no missing
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 0)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 1)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)
	MakeAggregatePrevoteAndVote(t, fixture, msgServer, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 3)

	fixture.OracleKeeper.UpdateExchangeRates(fixture.Ctx)

	// validators keep miss counters from last vote period
	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[0], 0))
	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[1], 0))
	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
	assert.Equal(t, uint64(1), fixture.OracleKeeper.MissCounters.GetOr(fixture.Ctx, ValAddrs[2], 0))
}
