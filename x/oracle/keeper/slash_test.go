package keeper

import (
	"testing"

	"cosmossdk.io/math"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

func TestSlashAndResetMissCounters(t *testing.T) {
	// initial setup
	input := CreateTestFixture(t)
	addr, val := ValAddrs[0], ValPubKeys[0]
	addr1, val1 := ValAddrs[1], ValPubKeys[1]
	amt := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	sh := stakingkeeper.NewMsgServerImpl(&input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := sh.CreateValidator(ctx, NewTestMsgCreateValidator(addr, val, amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(ctx, NewTestMsgCreateValidator(addr1, val1, amt))
	require.NoError(t, err)
	staking.EndBlocker(ctx, &input.StakingKeeper)

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

	votePeriodsPerWindow := math.LegacyNewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	slashFraction := input.OracleKeeper.SlashFraction(input.Ctx)
	minValidVotes := input.OracleKeeper.MinValidPerWindow(input.Ctx).MulInt64(votePeriodsPerWindow).Ceil().TruncateInt64()
	// Case 1, no slash
	input.OracleKeeper.MissCounters.Insert(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	staking.EndBlocker(input.Ctx, &input.StakingKeeper)

	validator, _ := input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.Equal(t, amt, validator.GetBondedTokens())

	// Case 2, slash
	input.OracleKeeper.MissCounters.Insert(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.Equal(t, amt.Sub(slashFraction.MulInt(amt).TruncateInt()), validator.GetBondedTokens())
	require.True(t, validator.IsJailed())

	// Case 3, slash unbonded validator
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	validator.Status = stakingtypes.Unbonded
	validator.Jailed = false
	validator.Tokens = amt
	input.StakingKeeper.SetValidator(input.Ctx, validator)

	input.OracleKeeper.MissCounters.Insert(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.Equal(t, amt, validator.Tokens)
	require.False(t, validator.IsJailed())

	// Case 4, slash jailed validator
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	validator.Status = stakingtypes.Bonded
	validator.Jailed = true
	validator.Tokens = amt
	input.StakingKeeper.SetValidator(input.Ctx, validator)

	input.OracleKeeper.MissCounters.Insert(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, _ = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.Equal(t, amt, validator.Tokens)
}

func TestInvalidVotesSlashing(t *testing.T) {
	input, h := Setup(t)
	params, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.ATOM, denoms.USD)}
	input.OracleKeeper.Params.Set(input.Ctx, params)
	input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, asset.Registry.Pair(denoms.ATOM, denoms.USD))

	votePeriodsPerWindow := math.LegacyNewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	slashFraction := input.OracleKeeper.SlashFraction(input.Ctx)
	minValidPerWindow := input.OracleKeeper.MinValidPerWindow(input.Ctx)

	for i := uint64(0); i < uint64(math.LegacyOneDec().Sub(minValidPerWindow).MulInt64(votePeriodsPerWindow).TruncateInt64()); i++ {
		input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

		// Account 1, govstable
		MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
			{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
		}, 0)

		// Account 2, govstable, miss vote
		MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
			{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate.Add(math.LegacyNewDec(100000000000000))},
		}, 1)

		// Account 3, govstable
		MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
			{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
		}, 2)

		// Account 4, govstable
		MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
			{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
		}, 3)

		input.OracleKeeper.UpdateExchangeRates(input.Ctx)
		// input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
		// input.OracleKeeper.UpdateExchangeRates(input.Ctx)

		require.Equal(t, i+1, input.OracleKeeper.MissCounters.GetOr(input.Ctx, ValAddrs[1], 0))
	}

	validator := input.StakingKeeper.Validator(input.Ctx, ValAddrs[1])
	require.Equal(t, testStakingAmt, validator.GetBondedTokens())

	// one more miss vote will inccur ValAddrs[1] slashing
	// Account 1, govstable
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
	}, 0)

	// Account 2, govstable, miss vote
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate.Add(math.LegacyNewDec(100000000000000))},
	}, 1)

	// Account 3, govstable
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
	}, 2)

	// Account 4, govstable
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate},
	}, 3)

	input.Ctx = input.Ctx.WithBlockHeight(votePeriodsPerWindow - 1)
	input.OracleKeeper.UpdateExchangeRates(input.Ctx)
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	// input.OracleKeeper.UpdateExchangeRates(input.Ctx)

	validator = input.StakingKeeper.Validator(input.Ctx, ValAddrs[1])
	require.Equal(t, math.LegacyOneDec().Sub(slashFraction).MulInt(testStakingAmt).TruncateInt(), validator.GetBondedTokens())
}

// TestWhitelistSlashing: Creates a scenario where one valoper (valIdx 0) does
// not vote throughout an entire vote window, while valopers 1 and 2 do.
func TestWhitelistSlashing(t *testing.T) {
	input, msgServer := Setup(t)

	votePeriodsPerSlashWindow := math.LegacyNewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	minValidVotePeriodsPerWindow := input.OracleKeeper.MinValidPerWindow(input.Ctx)

	pair := asset.Registry.Pair(denoms.ATOM, denoms.USD)
	priceVoteFromVal := func(valIdx int, block int64, erate sdk.Dec) {
		MakeAggregatePrevoteAndVote(t, input, msgServer, block,
			types.ExchangeRateTuples{{Pair: pair, ExchangeRate: erate}},
			valIdx)
	}
	input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, pair)
	perfs := input.OracleKeeper.UpdateExchangeRates(input.Ctx)
	require.EqualValues(t, 0, perfs.TotalRewardWeight())

	allowedMissPct := math.LegacyOneDec().Sub(minValidVotePeriodsPerWindow)
	allowedMissVotePeriods := allowedMissPct.MulInt64(votePeriodsPerSlashWindow).
		TruncateInt64()
	t.Logf("For %v blocks, valoper0 does not vote, while 1 and 2 do.", allowedMissVotePeriods)
	for idxMissPeriod := uint64(0); idxMissPeriod < uint64(allowedMissVotePeriods); idxMissPeriod++ {
		block := input.Ctx.BlockHeight() + 1
		input.Ctx = input.Ctx.WithBlockHeight(block)

		valIdx := 0 // Valoper doesn't vote (abstain)
		priceVoteFromVal(valIdx+1, block, testExchangeRate)
		priceVoteFromVal(valIdx+2, block, testExchangeRate)

		perfs := input.OracleKeeper.UpdateExchangeRates(input.Ctx)
		missCount := input.OracleKeeper.MissCounters.GetOr(input.Ctx, ValAddrs[0], 0)
		require.EqualValues(t, 0, missCount, perfs.String())
	}

	t.Log("valoper0 should not be slashed")
	validator := input.StakingKeeper.Validator(input.Ctx, ValAddrs[0])
	require.Equal(t, testStakingAmt, validator.GetBondedTokens())
}

func TestNotPassedBallotSlashing(t *testing.T) {
	input, h := Setup(t)
	params, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.ATOM, denoms.USD)}
	input.OracleKeeper.Params.Set(input.Ctx, params)

	// clear tobin tax to reset vote targets
	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[asset.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}
	input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, asset.Registry.Pair(denoms.ATOM, denoms.USD))

	input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

	// Account 1, govstable
	MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate}}, 0)

	input.OracleKeeper.UpdateExchangeRates(input.Ctx)
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	// input.OracleKeeper.UpdateExchangeRates(input.Ctx)
	require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, ValAddrs[0], 0))
	require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, ValAddrs[1], 0))
	require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, ValAddrs[2], 0))
}

func TestAbstainSlashing(t *testing.T) {
	input, h := Setup(t)

	// reset whitelisted pairs
	params, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)
	params.Whitelist = []asset.Pair{asset.Registry.Pair(denoms.ATOM, denoms.USD)}
	input.OracleKeeper.Params.Set(input.Ctx, params)
	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[asset.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}
	input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, asset.Registry.Pair(denoms.ATOM, denoms.USD))

	votePeriodsPerWindow := math.LegacyNewDec(int64(input.OracleKeeper.SlashWindow(input.Ctx))).QuoInt64(int64(input.OracleKeeper.VotePeriod(input.Ctx))).TruncateInt64()
	minValidPerWindow := input.OracleKeeper.MinValidPerWindow(input.Ctx)

	for i := uint64(0); i <= uint64(math.LegacyOneDec().Sub(minValidPerWindow).MulInt64(votePeriodsPerWindow).TruncateInt64()); i++ {
		input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

		// Account 1, ATOM/USD
		MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate}}, 0)

		// Account 2, ATOM/USD, abstain vote
		MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: math.LegacyOneDec().Neg()}}, 1)

		// Account 3, ATOM/USD
		MakeAggregatePrevoteAndVote(t, input, h, 0, types.ExchangeRateTuples{{Pair: asset.Registry.Pair(denoms.ATOM, denoms.USD), ExchangeRate: testExchangeRate}}, 2)

		input.OracleKeeper.UpdateExchangeRates(input.Ctx)
		input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
		// input.OracleKeeper.UpdateExchangeRates(input.Ctx)
		require.Equal(t, uint64(0), input.OracleKeeper.MissCounters.GetOr(input.Ctx, ValAddrs[1], 0))
	}

	validator := input.StakingKeeper.Validator(input.Ctx, ValAddrs[1])
	require.Equal(t, testStakingAmt, validator.GetBondedTokens())
}
