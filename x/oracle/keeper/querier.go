package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
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

	params, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

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
	exchangeRate, err := q.Keeper.GetExchangeRate(ctx, req.Pair)
	if err != nil {
		return nil, err
	}

	return &types.QueryExchangeRateResponse{ExchangeRate: exchangeRate}, nil
}

/*
Gets the time-weighted average price from ( ctx.BlockTime() - interval, ctx.BlockTime() ]
Note the open-ended right bracket.

If there's only one snapshot, then this function returns the price from that single snapshot.

Returns -1 if there's no price.
*/
func (q querier) ExchangeRateTwap(c context.Context, req *types.QueryExchangeRateRequest) (response *types.QueryExchangeRateResponse, err error) {
	if _, err = q.ExchangeRate(c, req); err != nil {
		return
	}

	ctx := sdk.UnwrapSDKContext(c)
	twap, err := q.Keeper.GetExchangeRateTwap(ctx, req.Pair)
	if err != nil {
		return &types.QueryExchangeRateResponse{}, err
	}
	return &types.QueryExchangeRateResponse{ExchangeRate: twap}, nil
}

// ExchangeRates queries exchange rates of all pairs
func (q querier) ExchangeRates(c context.Context, _ *types.QueryExchangeRatesRequest) (*types.QueryExchangeRatesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	var exchangeRates types.ExchangeRateTuples
	iter, err := q.Keeper.ExchangeRates.Iterate(ctx, &collections.Range[asset.Pair]{})
	if err != nil {
		q.Logger(ctx).Error("failed to iterate exchange rates", "error", err)
		return nil, err
	}
	kv, err := iter.KeyValues()
	if err != nil {
		q.Logger(ctx).Error("failed to get exchange rates key values", "error", err)
		return nil, err
	}

	for _, er := range kv {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{
			Pair:         er.Key,
			ExchangeRate: er.Value.ExchangeRate,
		})
	}

	return &types.QueryExchangeRatesResponse{ExchangeRates: exchangeRates}, nil
}

// Actives queries all pairs for which exchange rates exist
func (q querier) Actives(c context.Context, _ *types.QueryActivesRequest) (*types.QueryActivesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	iter, err := q.Keeper.ExchangeRates.Iterate(sdk.UnwrapSDKContext(c), &collections.Range[asset.Pair]{})
	if err != nil {
		q.Logger(ctx).Error("failed to iterate exchange rates", "error", err)
		return nil, err
	}
	keys, err := iter.Keys()
	if err != nil {
		q.Logger(ctx).Error("failed to get exchange rates keys", "error", err)
		return nil, err
	}
	return &types.QueryActivesResponse{Actives: keys}, nil
}

// VoteTargets queries the voting target list on current vote period
func (q querier) VoteTargets(c context.Context, _ *types.QueryVoteTargetsRequest) (*types.QueryVoteTargetsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryVoteTargetsResponse{VoteTargets: q.GetWhitelistedPairs(ctx)}, nil
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
	delegations, err := q.Keeper.FeederDelegations.Get(ctx, valAddr)
	if delegations == nil {
		delegations = sdk.AccAddress(valAddr)
	}
	return &types.QueryFeederDelegationResponse{
		FeederAddr: delegations.String(),
	}, nil
}

// MissCounter queries oracle miss counter of a validator
func (q querier) MissCounter(c context.Context, req *types.QueryMissCounterRequest) (*types.QueryMissCounterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	missCounter, err := q.MissCounters.Get(ctx, valAddr)
	if err != nil {
		missCounter = 0
	}
	return &types.QueryMissCounterResponse{
		MissCounter: missCounter,
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
	prevote, err := q.Prevotes.Get(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryAggregatePrevoteResponse{
		AggregatePrevote: prevote,
	}, nil
}

// AggregatePrevotes queries aggregate prevotes of all validators
func (q querier) AggregatePrevotes(c context.Context, _ *types.QueryAggregatePrevotesRequest) (*types.QueryAggregatePrevotesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	iter, err := q.Prevotes.Iterate(sdk.UnwrapSDKContext(c), &collections.Range[sdk.ValAddress]{})
	if err != nil {
		q.Logger(ctx).Error("failed to iterate prevotes", "error", err)
		return nil, err
	}
	values, err := iter.Values()
	if err != nil {
		q.Logger(ctx).Error("failed to get prevotes values", "error", err)
		return nil, err
	}
	return &types.QueryAggregatePrevotesResponse{AggregatePrevotes: values}, nil
}

// AggregateVote queries an aggregate vote of a validator
func (q querier) AggregateVote(c context.Context, req *types.QueryAggregateVoteRequest) (*types.QueryAggregateVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	vote, err := q.Keeper.Votes.Get(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryAggregateVoteResponse{
		AggregateVote: vote,
	}, nil
}

// AggregateVotes queries aggregate votes of all validators
func (q querier) AggregateVotes(c context.Context, _ *types.QueryAggregateVotesRequest) (*types.QueryAggregateVotesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	iter, err := q.Keeper.Votes.Iterate(sdk.UnwrapSDKContext(c), &collections.Range[sdk.ValAddress]{})
	if err != nil {
		q.Logger(ctx).Error("failed to iterate votes", "error", err)
		return nil, err
	}
	values, err := iter.Values()
	if err != nil {
		q.Logger(ctx).Error("failed to get votes values", "error", err)
		return nil, err
	}
	return &types.QueryAggregateVotesResponse{AggregateVotes: values}, nil
}
