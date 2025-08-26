package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/gastoken/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the x/gastoken keeper providing gRPC method
// handlers.
type Querier struct {
	Keeper
}

func NewQuerier(k Keeper) Querier {
	return Querier{Keeper: k}
}

func (q Querier) FeeTokens(ctx context.Context, _ *types.QueryFeeTokensRequest) (*types.QueryFeeTokensResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	feeTokens := q.Keeper.GetFeeTokens(sdkCtx)

	return &types.QueryFeeTokensResponse{FeeTokens: feeTokens}, nil
}

func (q Querier) Params(ctx context.Context, _ *types.ParamsRequest) (*types.ParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := q.Keeper.GetParams(sdkCtx)
	if err != nil {
		return nil, err
	}

	return &types.ParamsResponse{Params: params}, nil
}
