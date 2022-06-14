package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

func (k Keeper) Prices(goCtx context.Context, req *types.QueryPricesRequest) (*types.QueryPricesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var currentPrices types.CurrentPriceResponses
	for _, currentPrice := range k.GetCurrentPrices(ctx) {
		if currentPrice.PairID != "" {
			currentPrices = append(currentPrices, types.CurrentPriceResponse(currentPrice))
		}
	}

	return &types.QueryPricesResponse{
		Prices: currentPrices,
	}, nil
}
