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
	keeper.AllocateRewards(t, input, common.Pair_NIBI_NUSD.String(), sdk.NewCoins(rewards), periods)

	for i := uint64(1); i <= periods; i++ {
		for valIndex := 0; valIndex < validators; valIndex++ {
			// for doc's sake, this function is capable of making prevotes and votes because it
			// passes the current context block height for pre vote
			// then changes the height to current height + vote period for the vote
			makeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{
				Pair:         common.Pair_NIBI_NUSD.String(),
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

		require.Equalf(t, truncatedExpected, truncatedGot, "period: %d, %s <-> %s", i, truncatedExpected.String(), truncatedGot.String())
	}

	// assert there are no rewards for pair
	require.True(t, input.OracleKeeper.AccrueVotePeriodPairRewards(input.Ctx, common.Pair_NIBI_NUSD.String()).IsZero())

	// assert that there are no rewards instances
	require.Empty(t, input.OracleKeeper.PairRewards.Indexes.RewardsByPair.ExactMatch(input.Ctx, common.Pair_NIBI_NUSD.String()).PrimaryKeys())
}
