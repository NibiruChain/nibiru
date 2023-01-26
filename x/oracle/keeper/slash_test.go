package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	types2 "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/oracle"
	"github.com/NibiruChain/nibiru/x/oracle/keeper"
	types3 "github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestSlashAndResetMissCounters(t *testing.T) {
	// initial setup
	input := keeper.CreateTestInput(t)
	addr, val := keeper.ValAddrs[0], keeper.ValPubKeys[0]
	addr1, val1 := keeper.ValAddrs[1], keeper.ValPubKeys[1]
	amt := types.TokensFromConsensusPower(100, types.DefaultPowerReduction)
	sh := staking.NewHandler(input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := sh(ctx, keeper.NewTestMsgCreateValidator(addr, val, amt))
	require.NoError(t, err)
	_, err = sh(ctx, keeper.NewTestMsgCreateValidator(addr1, val1, amt))
	require.NoError(t, err)
	staking.EndBlocker(ctx, input.StakingKeeper)

	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, types.AccAddress(addr)),
		types.NewCoins(types.NewCoin(input.StakingKeeper.GetParams(ctx).BondDenom, keeper.InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, input.StakingKeeper.Validator(ctx, addr).GetBondedTokens())
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, types.AccAddress(addr1)),
		types.NewCoins(types.NewCoin(input.StakingKeeper.GetParams(ctx).BondDenom, keeper.InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, input.StakingKeeper.Validator(ctx, addr1).GetBondedTokens())

	votePeriodsPerWindow := types.NewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	slashFraction := input.OracleKeeper.SlashFraction(input.Ctx)
	minValidVotes := input.OracleKeeper.MinValidPerWindow(input.Ctx).MulInt64(votePeriodsPerWindow).TruncateInt64()
	// Case 1, no slash
	input.OracleKeeper.MissCounters.Insert(input.Ctx, keeper.ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	staking.EndBlocker(input.Ctx, input.StakingKeeper)

	validator, _ := input.StakingKeeper.GetValidator(input.Ctx, keeper.ValAddrs[0])
	require.Equal(t, amt, validator.GetBondedTokens())

	// Case 2, slash
	input.OracleKeeper.MissCounters.Insert(input.Ctx, keeper.ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, keeper.ValAddrs[0])
	require.Equal(t, amt.Sub(slashFraction.MulInt(amt).TruncateInt()), validator.GetBondedTokens())
	require.True(t, validator.IsJailed())

	// Case 3, slash unbonded validator
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, keeper.ValAddrs[0])
	validator.Status = types2.Unbonded
	validator.Jailed = false
	validator.Tokens = amt
	input.StakingKeeper.SetValidator(input.Ctx, validator)

	input.OracleKeeper.MissCounters.Insert(input.Ctx, keeper.ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, keeper.ValAddrs[0])
	require.Equal(t, amt, validator.Tokens)
	require.False(t, validator.IsJailed())

	// Case 4, slash jailed validator
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, keeper.ValAddrs[0])
	validator.Status = types2.Bonded
	validator.Jailed = true
	validator.Tokens = amt
	input.StakingKeeper.SetValidator(input.Ctx, validator)

	input.OracleKeeper.MissCounters.Insert(input.Ctx, keeper.ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, keeper.ValAddrs[0])
	require.Equal(t, amt, validator.Tokens)
}

func TestInvalidVotesSlashing(t *testing.T) {
	input, h := setup(t)
	params := input.OracleKeeper.GetParams(input.Ctx)
	params.Whitelist = []common.AssetPair{common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD)}
	input.OracleKeeper.SetParams(input.Ctx, params)
	input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD))

	votePeriodsPerWindow := types.NewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	slashFraction := input.OracleKeeper.SlashFraction(input.Ctx)
	minValidPerWindow := input.OracleKeeper.MinValidPerWindow(input.Ctx)

	for i := uint64(0); i < uint64(types.OneDec().Sub(minValidPerWindow).MulInt64(votePeriodsPerWindow).TruncateInt64()); i++ {
		input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

		// Account 1, govstable
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 0)

		// Account 2, govstable, miss vote
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate.Add(types.NewDec(100000000000000))}}, 1)

		// Account 3, govstable
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)

		oracle.EndBlocker(input.Ctx, input.OracleKeeper)
		require.Equal(t, i+1, input.OracleKeeper.MissCounters.GetOr(input.Ctx, keeper.ValAddrs[1], 0))
	}

	validator := input.StakingKeeper.Validator(input.Ctx, keeper.ValAddrs[1])
	require.Equal(t, stakingAmt, validator.GetBondedTokens())

	// one more miss vote will inccur keeper.ValAddrs[1] slashing
	// Account 1, govstable
	makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 0)

	// Account 2, govstable, miss vote
	makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate.Add(types.NewDec(100000000000000))}}, 1)

	// Account 3, govstable
	makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)

	input.Ctx = input.Ctx.WithBlockHeight(votePeriodsPerWindow - 1)
	oracle.EndBlocker(input.Ctx, input.OracleKeeper)
	validator = input.StakingKeeper.Validator(input.Ctx, keeper.ValAddrs[1])
	require.Equal(t, types.OneDec().Sub(slashFraction).MulInt(stakingAmt).TruncateInt(), validator.GetBondedTokens())
}

func TestWhitelistSlashing(t *testing.T) {
	input, h := setup(t)

	votePeriodsPerWindow := types.NewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	slashFraction := input.OracleKeeper.SlashFraction(input.Ctx)
	minValidPerWindow := input.OracleKeeper.MinValidPerWindow(input.Ctx)

	for i := uint64(0); i < uint64(types.OneDec().Sub(minValidPerWindow).MulInt64(votePeriodsPerWindow).TruncateInt64()); i++ {
		input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

		// Account 2, govstable
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 1)
		// Account 3, govstable
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)

		oracle.EndBlocker(input.Ctx, input.OracleKeeper)
		require.Equal(t, i+1, input.OracleKeeper.MissCounters.GetOr(input.Ctx, keeper.ValAddrs[0], 0))
	}

	validator := input.StakingKeeper.Validator(input.Ctx, keeper.ValAddrs[0])
	require.Equal(t, stakingAmt, validator.GetBondedTokens())

	// one more miss vote will inccur Account 1 slashing

	// Account 2, govstable
	makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 1)
	// Account 3, govstable
	makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)

	input.Ctx = input.Ctx.WithBlockHeight(votePeriodsPerWindow - 1)
	oracle.EndBlocker(input.Ctx, input.OracleKeeper)
	validator = input.StakingKeeper.Validator(input.Ctx, keeper.ValAddrs[0])
	require.Equal(t, types.OneDec().Sub(slashFraction).MulInt(stakingAmt).TruncateInt(), validator.GetBondedTokens())
}

func TestNotPassedBallotSlashing(t *testing.T) {
	input, h := setup(t)
	params := input.OracleKeeper.GetParams(input.Ctx)
	params.Whitelist = []common.AssetPair{common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD)}
	input.OracleKeeper.SetParams(input.Ctx, params)

	// clear tobin tax to reset vote targets
	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[common.AssetPair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}
	input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD))

	input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

	// Account 1, govstable
	makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 0)

	oracle.EndBlocker(input.Ctx, input.OracleKeeper)
	require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, keeper.ValAddrs[0], 0))
	require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, keeper.ValAddrs[1], 0))
	require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, keeper.ValAddrs[2], 0))
}

func TestAbstainSlashing(t *testing.T) {
	input, h := setup(t)
	params := input.OracleKeeper.GetParams(input.Ctx)
	params.Whitelist = []common.AssetPair{common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD)}
	input.OracleKeeper.SetParams(input.Ctx, params)

	// clear tobin tax to reset vote targets
	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[common.AssetPair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}
	input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD))

	votePeriodsPerWindow := types.NewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	minValidPerWindow := input.OracleKeeper.MinValidPerWindow(input.Ctx)

	for i := uint64(0); i <= uint64(types.OneDec().Sub(minValidPerWindow).MulInt64(votePeriodsPerWindow).TruncateInt64()); i++ {
		input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

		// Account 1, govstable
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 0)

		// Account 2, govstable, abstain vote
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: types.ZeroDec()}}, 1)

		// Account 3, govstable
		makeAggregatePrevoteAndVote(t, input, h, 0, types3.ExchangeRateTuples{{Pair: common.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), ExchangeRate: randomExchangeRate}}, 2)

		oracle.EndBlocker(input.Ctx, input.OracleKeeper)
		require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, keeper.ValAddrs[1], 0))
	}

	validator := input.StakingKeeper.Validator(input.Ctx, keeper.ValAddrs[1])
	require.Equal(t, stakingAmt, validator.GetBondedTokens())
}
