package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the x/FeeShare keeper providing gRPC method
// handlers.
type Querier struct {
	Keeper
}

func NewQuerier(k Keeper) Querier {
	return Querier{Keeper: k}
}

// Fee

// TODO FeeSharesAll returns all FeeShares that have been registered for fee
// distribution
func (q Querier) FeeShares(
	goCtx context.Context,
	req *types.QueryFeeSharesRequest,
) (*types.QueryFeeSharesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	iter := q.DevGasStore.Indexes.Deployer.ExactMatch(ctx, req.Deployer)
	return &types.QueryFeeSharesResponse{
		Feeshare: q.DevGasStore.Collect(ctx, iter),
	}, nil
}

// FeeShare returns the FeeShare that has been registered for fee distribution for a given
// contract
func (q Querier) FeeShare(
	goCtx context.Context,
	req *types.QueryFeeShareRequest,
) (*types.QueryFeeShareResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// check if the contract is a non-zero hex address
	contract, err := sdk.AccAddressFromBech32(req.ContractAddress)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid format for contract %s, should be bech32 ('nibi...')", req.ContractAddress,
		)
	}

	feeshare, found := q.GetFeeShare(ctx, contract)
	if !found {
		return nil, status.Errorf(
			codes.NotFound,
			"fees registered contract '%s'",
			req.ContractAddress,
		)
	}

	return &types.QueryFeeShareResponse{Feeshare: feeshare}, nil
}

// Params returns the fees module params
func (q Querier) Params(
	c context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// FeeSharesByWithdrawer returns all fees for a given withdraw address
func (q Querier) FeeSharesByWithdrawer( // nolint: dupl
	goCtx context.Context,
	req *types.QueryFeeSharesByWithdrawerRequest,
) (*types.QueryFeeSharesByWithdrawerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	iter := q.DevGasStore.Indexes.Withdrawer.ExactMatch(ctx, req.WithdrawerAddress)
	return &types.QueryFeeSharesByWithdrawerResponse{
		Feeshare: q.DevGasStore.Collect(ctx, iter),
	}, nil
}
