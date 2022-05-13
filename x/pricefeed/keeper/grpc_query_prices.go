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
	for _, cp := range k.GetCurrentPrices(ctx) {
		if cp.PairID != "" {
			currentPrices = append(currentPrices, types.CurrentPriceResponse(cp))
		}
	}

	return &types.QueryPricesResponse{
		Prices: currentPrices,
	}, nil
}
