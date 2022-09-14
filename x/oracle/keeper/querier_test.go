package keeper

import (
	"github.com/NibiruChain/nibiru/collections/keys"
	gogotypes "github.com/gogo/protobuf/types"
	"sort"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestQueryParams(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)

	querier := NewQuerier(input.OracleKeeper)
	res, err := querier.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	require.Equal(t, input.OracleKeeper.GetParams(input.Ctx), res.Params)
}

func TestQueryExchangeRate(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	rate := sdk.NewDec(1700)
	input.OracleKeeper.ExchangeRates.Insert(input.Ctx, keys.String(common.PairETHStable.String()), sdk.DecProto{Dec: rate})

	// empty request
	_, err := querier.ExchangeRate(ctx, nil)
	require.Error(t, err)

	// Query to grpc
	res, err := querier.ExchangeRate(ctx, &types.QueryExchangeRateRequest{
		Pair: common.PairETHStable.String(),
	})
	require.NoError(t, err)
	require.Equal(t, rate, res.ExchangeRate)
}

func TestQueryMissCounter(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	missCounter := uint64(1)
	input.OracleKeeper.MissCounters.Insert(input.Ctx, keys.String(ValAddrs[0].String()), gogotypes.UInt64Value{Value: missCounter})

	// empty request
	_, err := querier.MissCounter(ctx, nil)
	require.Error(t, err)

	// Query to grpc
	res, err := querier.MissCounter(ctx, &types.QueryMissCounterRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)
	require.Equal(t, missCounter, res.MissCounter)
}

func TestQueryExchangeRates(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	rate := sdk.NewDec(1700)
	input.OracleKeeper.ExchangeRates.Insert(input.Ctx, keys.String(common.PairBTCStable.String()), sdk.DecProto{Dec: rate})
	input.OracleKeeper.ExchangeRates.Insert(input.Ctx, keys.String(common.PairETHStable.String()), sdk.DecProto{Dec: rate})

	res, err := querier.ExchangeRates(ctx, &types.QueryExchangeRatesRequest{})
	require.NoError(t, err)

	require.Equal(t, types.ExchangeRateTuples{
		{Pair: common.PairBTCStable.String(), ExchangeRate: rate},
		{Pair: common.PairETHStable.String(), ExchangeRate: rate},
	}, res.ExchangeRates)
}

func TestQueryActives(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	rate := sdk.NewDec(1700)
	input.OracleKeeper.ExchangeRates.Insert(input.Ctx, keys.String(common.PairBTCStable.String()), sdk.DecProto{Dec: rate})
	input.OracleKeeper.ExchangeRates.Insert(input.Ctx, keys.String(common.PairGovStable.String()), sdk.DecProto{Dec: rate})
	input.OracleKeeper.ExchangeRates.Insert(input.Ctx, keys.String(common.PairETHStable.String()), sdk.DecProto{Dec: rate})

	res, err := querier.Actives(ctx, &types.QueryActivesRequest{})
	require.NoError(t, err)

	targetDenoms := []string{
		common.PairBTCStable.String(),
		common.PairETHStable.String(),
		common.PairGovStable.String(),
	}

	require.Equal(t, targetDenoms, res.Actives)
}

func TestQueryFeederDelegation(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	input.OracleKeeper.FeederDelegations.Insert(input.Ctx, keys.String(ValAddrs[0].String()), gogotypes.BytesValue{Value: Addrs[1]})

	// empty request
	_, err := querier.FeederDelegation(ctx, nil)
	require.Error(t, err)

	res, err := querier.FeederDelegation(ctx, &types.QueryFeederDelegationRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)

	require.Equal(t, Addrs[1].String(), res.FeederAddr)
}

func TestQueryAggregatePrevote(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	prevote1 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[0], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, keys.String(ValAddrs[0].String()), prevote1)
	prevote2 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[1], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, keys.String(ValAddrs[1].String()), prevote2)

	// validator 0 address params
	res, err := querier.AggregatePrevote(ctx, &types.QueryAggregatePrevoteRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)
	require.Equal(t, prevote1, res.AggregatePrevote)

	// empty request
	_, err = querier.AggregatePrevote(ctx, nil)
	require.Error(t, err)

	// validator 1 address params
	res, err = querier.AggregatePrevote(ctx, &types.QueryAggregatePrevoteRequest{
		ValidatorAddr: ValAddrs[1].String(),
	})
	require.NoError(t, err)
	require.Equal(t, prevote2, res.AggregatePrevote)
}

func TestQueryAggregatePrevotes(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	prevote1 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[0], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, keys.String(ValAddrs[0].String()), prevote1)
	prevote2 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[1], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, keys.String(ValAddrs[1].String()), prevote2)
	prevote3 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[2], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, keys.String(ValAddrs[2].String()), prevote3)

	expectedPrevotes := []types.AggregateExchangeRatePrevote{prevote1, prevote2, prevote3}
	sort.SliceStable(expectedPrevotes, func(i, j int) bool {
		return expectedPrevotes[i].Voter < expectedPrevotes[j].Voter
	})

	res, err := querier.AggregatePrevotes(ctx, &types.QueryAggregatePrevotesRequest{})
	require.NoError(t, err)
	require.Equal(t, expectedPrevotes, res.AggregatePrevotes)
}

func TestQueryAggregateVote(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	vote1 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdk.OneDec()}}, ValAddrs[0])
	input.OracleKeeper.Votes.Insert(input.Ctx, keys.String(ValAddrs[0].String()), vote1)
	vote2 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdk.OneDec()}}, ValAddrs[1])
	input.OracleKeeper.Votes.Insert(input.Ctx, keys.String(ValAddrs[1].String()), vote2)

	// empty request
	_, err := querier.AggregateVote(ctx, nil)
	require.Error(t, err)

	// validator 0 address params
	res, err := querier.AggregateVote(ctx, &types.QueryAggregateVoteRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)
	require.Equal(t, vote1, res.AggregateVote)

	// validator 1 address params
	res, err = querier.AggregateVote(ctx, &types.QueryAggregateVoteRequest{
		ValidatorAddr: ValAddrs[1].String(),
	})
	require.NoError(t, err)
	require.Equal(t, vote2, res.AggregateVote)
}

func TestQueryAggregateVotes(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	vote1 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdk.OneDec()}}, ValAddrs[0])
	input.OracleKeeper.Votes.Insert(input.Ctx, keys.String(ValAddrs[0].String()), vote1)
	vote2 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdk.OneDec()}}, ValAddrs[1])
	input.OracleKeeper.Votes.Insert(input.Ctx, keys.String(ValAddrs[1].String()), vote2)
	vote3 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdk.OneDec()}}, ValAddrs[2])
	input.OracleKeeper.Votes.Insert(input.Ctx, keys.String(ValAddrs[2].String()), vote3)

	expectedVotes := []types.AggregateExchangeRateVote{vote1, vote2, vote3}
	sort.SliceStable(expectedVotes, func(i, j int) bool {
		return expectedVotes[i].Voter < expectedVotes[j].Voter
	})

	res, err := querier.AggregateVotes(ctx, &types.QueryAggregateVotesRequest{})
	require.NoError(t, err)
	require.Equal(t, expectedVotes, res.AggregateVotes)
}

func TestQueryVoteTargets(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	// clear pairs
	input.OracleKeeper.ClearPairs(input.Ctx)

	voteTargets := []string{"denom", "denom2", "denom3"}
	for _, target := range voteTargets {
		input.OracleKeeper.SetPair(input.Ctx, target)
	}

	res, err := querier.VoteTargets(ctx, &types.QueryVoteTargetsRequest{})
	require.NoError(t, err)
	require.Equal(t, voteTargets, res.VoteTargets)
}
