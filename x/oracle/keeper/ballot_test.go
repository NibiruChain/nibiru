package keeper

import (
	"sort"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestOrganizeAggregate(t *testing.T) {
	input := CreateTestInput(t)

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
		types.NewExchangeRateBallot(sdk.NewDec(17), common.Pair_BTC_NUSD.String(), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(10), common.Pair_BTC_NUSD.String(), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(6), common.Pair_BTC_NUSD.String(), ValAddrs[2], power),
	}
	ethBallot := types.ExchangeRateBallots{
		types.NewExchangeRateBallot(sdk.NewDec(1000), common.Pair_ETH_NUSD.String(), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(1300), common.Pair_ETH_NUSD.String(), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(2000), common.Pair_ETH_NUSD.String(), ValAddrs[2], power),
	}

	for i := range btcBallot {
		input.OracleKeeper.Votes.Insert(
			input.Ctx,
			ValAddrs[i],
			types.NewAggregateExchangeRateVote(
				types.ExchangeRateTuples{
					{Pair: btcBallot[i].Pair, ExchangeRate: btcBallot[i].ExchangeRate},
					{Pair: ethBallot[i].Pair, ExchangeRate: ethBallot[i].ExchangeRate},
				},
				ValAddrs[i],
			),
		)
	}

	// organize votes by pair
	ballotMap := input.OracleKeeper.groupBallotsByPair(input.Ctx, map[string]types.ValidatorPerformance{
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
	sort.Sort(btcBallot)
	sort.Sort(ethBallot)
	sort.Sort(ballotMap[common.Pair_BTC_NUSD.String()])
	sort.Sort(ballotMap[common.Pair_ETH_NUSD.String()])

	require.Equal(t, btcBallot, ballotMap[common.Pair_BTC_NUSD.String()])
	require.Equal(t, ethBallot, ballotMap[common.Pair_ETH_NUSD.String()])
}

func TestClearBallots(t *testing.T) {
	input := CreateTestInput(t)

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
		types.NewExchangeRateBallot(sdk.NewDec(17), common.Pair_BTC_NUSD.String(), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(10), common.Pair_BTC_NUSD.String(), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(6), common.Pair_BTC_NUSD.String(), ValAddrs[2], power),
	}
	ethBallot := types.ExchangeRateBallots{
		types.NewExchangeRateBallot(sdk.NewDec(1000), common.Pair_ETH_NUSD.String(), ValAddrs[0], power),
		types.NewExchangeRateBallot(sdk.NewDec(1300), common.Pair_ETH_NUSD.String(), ValAddrs[1], power),
		types.NewExchangeRateBallot(sdk.NewDec(2000), common.Pair_ETH_NUSD.String(), ValAddrs[2], power),
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

func TestApplyWhitelist(t *testing.T) {
	input := CreateTestInput(t)

	whitelist := []string{
		"nibi:usd",
		"btc:usd",
	}

	// prepare test by resetting the genesis pairs
	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[string]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}
	for _, p := range whitelist {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, p)
	}

	voteTargets := map[string]struct{}{
		"nibi:usd": {},
		"btc:usd":  {},
	}
	// no updates case
	input.OracleKeeper.updateWhitelist(input.Ctx, whitelist, voteTargets)

	var gotPairs []string
	gotPairs = append(gotPairs, input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[string]{}).Keys()...)

	sort.Slice(whitelist, func(i, j int) bool {
		return whitelist[i] < whitelist[j]
	})
	require.Equal(t, whitelist, gotPairs)

	// len update (fast path)
	whitelist = append(whitelist, "nibi:eth")
	input.OracleKeeper.updateWhitelist(input.Ctx, whitelist, voteTargets)

	gotPairs = []string{}
	gotPairs = append(gotPairs, input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[string]{}).Keys()...)

	sort.Slice(whitelist, func(i, j int) bool {
		return whitelist[i] < whitelist[j]
	})
	require.Equal(t, whitelist, gotPairs)

	// diff update (slow path)
	voteTargets["nibi:eth"] = struct{}{} // add previous pair
	whitelist[0] = "nibi:usdt"           // update first pair
	input.OracleKeeper.updateWhitelist(input.Ctx, whitelist, voteTargets)

	gotPairs = []string{}
	gotPairs = append(gotPairs, input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[string]{}).Keys()...)

	sort.Slice(whitelist, func(i, j int) bool {
		return whitelist[i] < whitelist[j]
	})
	require.Equal(t, whitelist, gotPairs)
}
