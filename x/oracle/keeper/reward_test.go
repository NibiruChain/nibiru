package keeper

import (
	"github.com/NibiruChain/nibiru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeeper_PairRewards(t *testing.T) {
	i := CreateTestInput(t)
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

// Test a reward giving mechanism
/* TODO(mercilex): currently not appliable https://github.com/NibiruChain/nibiru/issues/805
func TestRewardBallotWinners(t *testing.T) {
	// initial setup
	input := CreateTestInput(t)
	addr, val := ValAddrs[0], ValPubKeys[0]
	addr1, val1 := ValAddrs[1], ValPubKeys[1]
	amt := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	sh := staking.NewHandler(input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := sh(ctx, NewTestMsgCreateValidator(addr, val, amt))
	require.NoError(t, err)
	_, err = sh(ctx, NewTestMsgCreateValidator(addr1, val1, amt))
	require.NoError(t, err)
	staking.EndBlocker(ctx, input.StakingKeeper)

	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr)),
		sdk.NewCoins(sdk.NewCoin(input.StakingKeeper.GetParams(ctx).BondDenom, InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, input.StakingKeeper.Validator(ctx, addr).GetBondedTokens())
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr1)),
		sdk.NewCoins(sdk.NewCoin(input.StakingKeeper.GetParams(ctx).BondDenom, InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, input.StakingKeeper.Validator(ctx, addr1).GetBondedTokens())

	// Add claim pools
	claim := types.NewClaim(10, 10, 0, addr)
	claim2 := types.NewClaim(20, 20, 0, addr1)
	claims := map[string]types.Claim{
		addr.String():  claim,
		addr1.String(): claim2,
	}

	// Prepare reward pool
	givingAmt := sdk.NewCoins(sdk.NewInt64Coin(common.DenomGov, 30000000), sdk.NewInt64Coin(core.MicroUSDDenom, 40000000))
	acc := input.AccountKeeper.GetModuleAccount(ctx, types.ModuleName)
	err = FundAccount(input, acc.GetAddress(), givingAmt)
	require.NoError(t, err)

	voteTargets := make(map[string]sdk.Dec)
	input.OracleKeeper.IteratePairs(ctx, func(denom string, tobinTax sdk.Dec) bool {
		voteTargets[denom] = tobinTax
		return false
	})

	votePeriodsPerWindow := sdk.NewDec((int64)(input.OracleKeeper.RewardDistributionWindow(input.Ctx))).
		QuoInt64((int64)(input.OracleKeeper.VotePeriod(input.Ctx))).
		TruncateInt64()
	input.OracleKeeper.RewardBallotWinners(ctx, (int64)(input.OracleKeeper.VotePeriod(input.Ctx)), (int64)(input.OracleKeeper.RewardDistributionWindow(input.Ctx)), voteTargets, claims)
	outstandingRewardsDec := input.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, addr)
	outstandingRewards, _ := outstandingRewardsDec.TruncateDecimal()
	require.Equal(t, sdk.NewDecFromInt(givingAmt.AmountOf(common.DenomGov)).QuoInt64(votePeriodsPerWindow).QuoInt64(3).TruncateInt(),
		outstandingRewards.AmountOf(common.DenomGov))
	require.Equal(t, sdk.NewDecFromInt(givingAmt.AmountOf(core.MicroUSDDenom)).QuoInt64(votePeriodsPerWindow).QuoInt64(3).TruncateInt(),
		outstandingRewards.AmountOf(core.MicroUSDDenom))

	outstandingRewardsDec1 := input.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, addr1)
	outstandingRewards1, _ := outstandingRewardsDec1.TruncateDecimal()
	require.Equal(t, sdk.NewDecFromInt(givingAmt.AmountOf(common.DenomGov)).QuoInt64(votePeriodsPerWindow).QuoInt64(3).MulInt64(2).TruncateInt(),
		outstandingRewards1.AmountOf(common.DenomGov))
	require.Equal(t, sdk.NewDecFromInt(givingAmt.AmountOf(core.MicroUSDDenom)).QuoInt64(votePeriodsPerWindow).QuoInt64(3).MulInt64(2).TruncateInt(),
		outstandingRewards1.AmountOf(core.MicroUSDDenom))
}
*/
