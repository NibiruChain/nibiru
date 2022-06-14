package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

func (k Keeper) Pairs(goCtx context.Context, req *types.QueryPairsRequest) (*types.QueryPairsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var pairs types.PairResponses
	for _, pair := range k.GetPairs(ctx) {
		pairs = append(pairs, pair.ToPairResponse())
	}

	return &types.QueryPairsResponse{
		Pairs: pairs,
	}, nil
}
