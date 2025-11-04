package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/x/nutil/asset"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// Params queries params of distribution module
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	var params types.Params

	params, err := k.ModuleParams.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}

// ExchangeRate queries exchange rate of a pair
func (k Keeper) ExchangeRate(c context.Context, req *types.QueryExchangeRateRequest) (*types.QueryExchangeRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if len(req.Pair) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty pair")
	}

	ctx := sdk.UnwrapSDKContext(c)
	out, err := k.ExchangeRateMap.Get(ctx, req.Pair)
	if err != nil {
		return nil, err
	}

	var isVintage bool // if the exchange rate has passed its expiration block
	oracleParams, err := k.ModuleParams.Get(ctx)
	if err != nil {
		isVintage = false
	} else {
		expirationBlock := out.CreatedBlock + oracleParams.ExpirationBlocks
		isVintage = expirationBlock <= uint64(ctx.BlockHeight())
	}

	return &types.QueryExchangeRateResponse{
		ExchangeRate:     out.ExchangeRate,
		BlockTimestampMs: out.BlockTimestampMs,
		BlockHeight:      out.CreatedBlock,
		IsVintage:        isVintage,
	}, nil
}

/*
Gets the time-weighted average price from ( ctx.BlockTime() - interval, ctx.BlockTime() ]
Note the open-ended right bracket.

If there's only one snapshot, then this function returns the price from that single snapshot.

Returns -1 if there's no price.
*/
func (k Keeper) ExchangeRateTwap(c context.Context, req *types.QueryExchangeRateRequest) (response *types.QueryExchangeRateResponse, err error) {
	if _, err = k.ExchangeRate(c, req); err != nil {
		return
	}

	ctx := sdk.UnwrapSDKContext(c)
	twap, err := k.GetExchangeRateTwap(ctx, req.Pair)
	if err != nil {
		return &types.QueryExchangeRateResponse{}, err
	}
	return &types.QueryExchangeRateResponse{ExchangeRate: twap}, nil
}

// ExchangeRates queries exchange rates of all pairs
func (k Keeper) ExchangeRates(c context.Context, _ *types.QueryExchangeRatesRequest) (*types.QueryExchangeRatesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	var exchangeRates types.ExchangeRateTuples
	for _, er := range k.ExchangeRateMap.Iterate(ctx, collections.Range[asset.Pair]{}).KeyValues() {
		exchangeRates = append(exchangeRates, types.ExchangeRateTuple{
			Pair:         er.Key,
			ExchangeRate: er.Value.ExchangeRate,
		})
	}

	return &types.QueryExchangeRatesResponse{ExchangeRates: exchangeRates}, nil
}

// Actives queries all pairs for which exchange rates exist
func (k Keeper) Actives(c context.Context, _ *types.QueryActivesRequest) (*types.QueryActivesResponse, error) {
	return &types.QueryActivesResponse{Actives: k.ExchangeRateMap.Iterate(sdk.UnwrapSDKContext(c), collections.Range[asset.Pair]{}).Keys()}, nil
}

// VoteTargets queries the voting target list on current vote period
func (k Keeper) VoteTargets(c context.Context, _ *types.QueryVoteTargetsRequest) (*types.QueryVoteTargetsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryVoteTargetsResponse{VoteTargets: k.GetWhitelistedPairs(ctx)}, nil
}

// FeederDelegation queries the account address that the validator operator delegated oracle vote rights to
func (k Keeper) FeederDelegation(c context.Context, req *types.QueryFeederDelegationRequest) (*types.QueryFeederDelegationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryFeederDelegationResponse{
		FeederAddr: k.FeederDelegations.GetOr(ctx, valAddr, sdk.AccAddress(valAddr)).String(),
	}, nil
}

// MissCounter queries oracle miss counter of a validator
func (k Keeper) MissCounter(
	c context.Context,
	req *types.QueryMissCounterRequest,
) (*types.QueryMissCounterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryMissCounterResponse{
		MissCounter: k.MissCounters.GetOr(ctx, valAddr, 0),
	}, nil
}

// AggregatePrevote queries an aggregate prevote of a validator
func (k Keeper) AggregatePrevote(
	c context.Context,
	req *types.QueryAggregatePrevoteRequest,
) (*types.QueryAggregatePrevoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	prevote, err := k.Prevotes.Get(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryAggregatePrevoteResponse{
		AggregatePrevote: prevote,
	}, nil
}

// AggregatePrevotes queries aggregate prevotes of all validators
func (k Keeper) AggregatePrevotes(
	c context.Context,
	_ *types.QueryAggregatePrevotesRequest,
) (*types.QueryAggregatePrevotesResponse, error) {
	return &types.QueryAggregatePrevotesResponse{AggregatePrevotes: k.Prevotes.Iterate(sdk.UnwrapSDKContext(c), collections.Range[sdk.ValAddress]{}).Values()}, nil
}

// AggregateVote queries an aggregate vote of a validator
func (k Keeper) AggregateVote(
	c context.Context,
	req *types.QueryAggregateVoteRequest,
) (*types.QueryAggregateVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	vote, err := k.Votes.Get(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryAggregateVoteResponse{
		AggregateVote: vote,
	}, nil
}

// AggregateVotes queries aggregate votes of all validators
func (k Keeper) AggregateVotes(
	c context.Context,
	_ *types.QueryAggregateVotesRequest,
) (*types.QueryAggregateVotesResponse, error) {
	return &types.QueryAggregateVotesResponse{
		AggregateVotes: k.Votes.Iterate(
			sdk.UnwrapSDKContext(c),
			collections.Range[sdk.ValAddress]{},
		).Values(),
	}, nil
}
