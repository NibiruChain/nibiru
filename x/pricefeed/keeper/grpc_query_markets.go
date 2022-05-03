package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Pairs(goCtx context.Context, req *types.QueryPairsRequest) (*types.QueryPairsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var markets types.PairResponses
	for _, market := range k.GetPairs(ctx) {
		markets = append(markets, market.ToPairResponse())
	}

	return &types.QueryPairsResponse{
		Pairs: markets,
	}, nil
}
