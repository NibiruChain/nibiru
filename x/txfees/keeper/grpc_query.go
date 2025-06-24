package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the x/txfees keeper providing gRPC method
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

func (q Querier) BaseDenom(ctx context.Context, _ *types.QueryBaseDenomRequest) (*types.QueryBaseDenomResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	baseDenom, err := q.Keeper.GetBaseDenom(sdkCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryBaseDenomResponse{BaseDenom: baseDenom}, nil
}
