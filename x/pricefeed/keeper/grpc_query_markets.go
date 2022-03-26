package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Markets(goCtx context.Context, req *types.QueryMarketsRequest) (*types.QueryMarketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var markets types.MarketResponses
	for _, market := range k.GetMarkets(ctx) {
		markets = append(markets, market.ToMarketResponse())
	}

	return &types.QueryMarketsResponse{
		Markets: markets,
	}, nil
}
