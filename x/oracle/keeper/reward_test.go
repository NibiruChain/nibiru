package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/oracle"
	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestKeeper_PairRewards(t *testing.T) {
	i := keeper.CreateTestInput(t)
	k, ctx := i.OracleKeeper, i.Ctx

	t.Run("get, set, delete", func(t *testing.T) {
		// test delete not found
		err := k.DeletePairReward(ctx, "idk", 0)
		require.Error(t, err)
		// test set and get (found)
		reward := &types.PairReward{
			Pair:        "BTC-USD",
			Id:          10,
			VotePeriods: 100,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 1000)),
		}
		k.SetPairReward(ctx, reward)
		gotReward, err := k.GetPairReward(ctx, reward.Pair, reward.Id)
		require.NoError(t, err)
		require.Equal(t, reward, gotReward)
		// test delete and get (not found)
		err = k.DeletePairReward(ctx, reward.Pair, reward.Id)
		require.NoError(t, err)
		_, err = k.GetPairReward(ctx, "idk", 0)
		require.Error(t, err)
	})

	t.Run("next key", func(t *testing.T) {
		key := k.NextPairRewardKey(ctx)
		require.Equal(t, key, uint64(0))
		key = k.NextPairRewardKey(ctx)
		require.Equal(t, key, uint64(1))
	})

	t.Run("iterations", func(t *testing.T) {
		reward1 := &types.PairReward{
			Pair:        "BTC:USD",
			VotePeriods: 100,
			Coins:       sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(1000))),
		}
		k.CreatePairReward(ctx, reward1)

		reward2 := &types.PairReward{
			Pair:        "ETH:USD",
			VotePeriods: 50,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 100)),
		}
		k.CreatePairReward(ctx, reward2)

		reward3 := &types.PairReward{
			Pair:        "BTC:USD",
			VotePeriods: 30,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 200), sdk.NewInt64Coin("nibi", 1000)),
		}
		k.CreatePairReward(ctx, reward3)

		// this tests prefix overlaps
		reward4 := &types.PairReward{
			Pair:        "BTC:USDT",
			VotePeriods: 3,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 200)),
		}
		k.CreatePairReward(ctx, reward4)

		// iterate by pair
		expected := []*types.PairReward{reward1, reward3}
		got := []*types.PairReward{}
		k.IteratePairRewards(ctx, reward1.Pair, func(rewards *types.PairReward) (stop bool) {
			got = append(got, rewards)
			return false
		})
		require.Equal(t, expected, got)

		// iterate all
		expected = []*types.PairReward{reward1, reward3, reward4, reward2}
		got = nil
		k.IterateAllPairRewards(ctx, func(rewards *types.PairReward) (stop bool) {
			got = append(got, rewards)
			return false
		})
		require.Equal(t, expected, got)
	})
}

func TestKeeper_RewardsDistributionMultiVotePeriods(t *testing.T) {
	// this simulates allocating rewards for the pair gov stable
	// over 5 voting periods. It simulates rewards are correctly
	// distributed over 5 voting periods to 5 validators.
	// then we simulate that after the 5 voting periods are
	// finished no more rewards distribution happen.
	const periods uint64 = 5
	const validators = 3
	input, h := setup(t) // set vote threshold
	params := input.OracleKeeper.GetParams(input.Ctx)
	input.OracleKeeper.SetParams(input.Ctx, params)

	rewards := sdk.NewInt64Coin("reward", 1_000_000)
	valPeriodicRewards := sdk.NewDecCoinsFromCoins(rewards).
		QuoDec(sdk.NewDec(int64(periods))).
		QuoDec(sdk.NewDec(int64(validators)))
	keeper.AllocateRewards(t, input, common.PairGovStable.String(), sdk.NewCoins(rewards), periods)

	for i := uint64(1); i <= periods; i++ {
		for valIndex := 0; valIndex < validators; valIndex++ {
			// for doc's sake, this function is capable of making prevotes and votes because it
			// passes the current context block height for pre vote
			// then changes the height to current height + vote period for the vote
			makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{
				Pair:         common.PairGovStable.String(),
				ExchangeRate: randomExchangeRate,
			}}, valIndex)
		}
		input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)
		oracle.EndBlocker(input.Ctx, input.OracleKeeper)
		// assert rewards
		distributionRewards := input.DistrKeeper.GetValidatorOutstandingRewards(input.Ctx.WithBlockHeight(input.Ctx.BlockHeight()+1), keeper.ValAddrs[0])
		truncatedGot, _ := distributionRewards.Rewards.
			QuoDec(sdk.NewDec(int64(i))). // outstanding rewards will count for the previous vote period too, so we divide it by current period
			TruncateDecimal()             // NOTE: not applying this on truncatedExpected because of rounding the test fails
		truncatedExpected, _ := valPeriodicRewards.TruncateDecimal()

		require.Equalf(t, truncatedExpected, truncatedGot, "period: %d, %s <-> %s", i, truncatedExpected, truncatedGot)
	}

	// assert there are no rewards for pair
	require.True(t, input.OracleKeeper.AccrueVotePeriodPairRewards(input.Ctx, common.PairGovStable.String()).IsZero())

	// assert that there are no rewards instances
	found := false
	input.OracleKeeper.IteratePairRewards(input.Ctx, common.PairGovStable.String(), func(rewards *types.PairReward) (stop bool) {
		found = true
		return true
	})
	require.False(t, found)
}
