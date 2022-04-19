package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Price(goCtx context.Context, req *types.QueryPriceRequest) (*types.QueryPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	_, found := k.GetPair(ctx, req.PairId)
	if !found {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}
	tokens := common.DenomsFromPoolName(req.PairId)
	token0, token1 := tokens[0], tokens[1]
	currentPrice, sdkErr := k.GetCurrentPrice(ctx, token0, token1)
	if sdkErr != nil {
		return nil, sdkErr
	}

	return &types.QueryPriceResponse{
		Price: types.CurrentPriceResponse{PairID: req.PairId, Price: currentPrice.Price},
	}, nil
}
