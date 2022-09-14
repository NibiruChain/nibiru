package keeper

import (
	"context"
	"github.com/NibiruChain/nibiru/collections/keys"
	gogotypes "github.com/gogo/protobuf/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over q
type querier struct {
	Keeper
}

// NewQuerier returns an implementation of the oracle QueryServer interface
// for the provided Keeper.
func NewQuerier(keeper Keeper) types.QueryServer {
	return &querier{Keeper: keeper}
}

var _ types.QueryServer = querier{}

// Params queries params of distribution module
func (q querier) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	var params types.Params
	q.paramSpace.GetParamSet(ctx, &params)

	return &types.QueryParamsResponse{Params: params}, nil
}

// ExchangeRate queries exchange rate of a pair
func (q querier) ExchangeRate(c context.Context, req *types.QueryExchangeRateRequest) (*types.QueryExchangeRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if len(req.Pair) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty pair")
	}

	ctx := sdk.UnwrapSDKContext(c)
	exchangeRate, err := q.Keeper.ExchangeRates.Get(ctx, keys.String(req.Pair))
	if err != nil {
		return nil, err
	}

	return &types.QueryExchangeRateResponse{ExchangeRate: exchangeRate.Dec}, nil
}

// ExchangeRates queries exchange rates of all pairs
func (q querier) ExchangeRates(c context.Context, _ *types.QueryExchangeRatesRequest) (*types.QueryExchangeRatesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	var exchangeRates types.ExchangeRateTuples
	for _, er := range q.Keeper.ExchangeRates.Iterate(ctx, keys.NewRange[keys.StringKey]()).KeyValues() {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{
			Pair:         string(er.Key),
			ExchangeRate: er.Value.Dec,
		})
	}

	return &types.QueryExchangeRatesResponse{ExchangeRates: exchangeRates}, nil
}

// Actives queries all pairs for which exchange rates exist
func (q querier) Actives(c context.Context, _ *types.QueryActivesRequest) (*types.QueryActivesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	var pairs []string
	for _, er := range q.Keeper.ExchangeRates.Iterate(ctx, keys.NewRange[keys.StringKey]()).Keys() {
		pairs = append(pairs, string(er))
	}

	return &types.QueryActivesResponse{Actives: pairs}, nil
}

// VoteTargets queries the voting target list on current vote period
func (q querier) VoteTargets(c context.Context, _ *types.QueryVoteTargetsRequest) (*types.QueryVoteTargetsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryVoteTargetsResponse{VoteTargets: q.GetVoteTargets(ctx)}, nil
}

// FeederDelegation queries the account address that the validator operator delegated oracle vote rights to
func (q querier) FeederDelegation(c context.Context, req *types.QueryFeederDelegationRequest) (*types.QueryFeederDelegationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryFeederDelegationResponse{
		FeederAddr: sdk.AccAddress(q.FeederDelegations.GetOr(ctx, keys.String(req.ValidatorAddr), gogotypes.BytesValue{Value: valAddr}).Value).String(),
	}, nil
}

// MissCounter queries oracle miss counter of a validator
func (q querier) MissCounter(c context.Context, req *types.QueryMissCounterRequest) (*types.QueryMissCounterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	_, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryMissCounterResponse{
		MissCounter: q.MissCounters.GetOr(ctx, keys.String(req.ValidatorAddr), gogotypes.UInt64Value{Value: 0}).Value,
	}, nil
}

// AggregatePrevote queries an aggregate prevote of a validator
func (q querier) AggregatePrevote(c context.Context, req *types.QueryAggregatePrevoteRequest) (*types.QueryAggregatePrevoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	prevote, err := q.Keeper.Prevotes.Get(ctx, keys.String(valAddr.String()))
	if err != nil {
		return nil, err
	}

	return &types.QueryAggregatePrevoteResponse{
		AggregatePrevote: prevote,
	}, nil
}

// AggregatePrevotes queries aggregate prevotes of all validators
func (q querier) AggregatePrevotes(goCtx context.Context, _ *types.QueryAggregatePrevotesRequest) (*types.QueryAggregatePrevotesResponse, error) {
	return &types.QueryAggregatePrevotesResponse{
		AggregatePrevotes: q.Keeper.Prevotes.Iterate(sdk.UnwrapSDKContext(goCtx), keys.NewRange[keys.StringKey]()).Values(),
	}, nil
}

// AggregateVote queries an aggregate vote of a validator
func (q querier) AggregateVote(c context.Context, req *types.QueryAggregateVoteRequest) (*types.QueryAggregateVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	_, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	vote, err := q.Votes.Get(ctx, keys.String(req.ValidatorAddr))
	if err != nil {
		return nil, err
	}

	return &types.QueryAggregateVoteResponse{
		AggregateVote: vote,
	}, nil
}

// AggregateVotes queries aggregate votes of all validators
func (q querier) AggregateVotes(c context.Context, _ *types.QueryAggregateVotesRequest) (*types.QueryAggregateVotesResponse, error) {

	return &types.QueryAggregateVotesResponse{
		AggregateVotes: q.Keeper.Votes.Iterate(sdk.UnwrapSDKContext(c), keys.NewRange[keys.StringKey]()).Values(),
	}, nil
}
