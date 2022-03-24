package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) RawPrices(goCtx context.Context, req *types.QueryRawPricesRequest) (*types.QueryRawPricesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	_, found := k.GetMarket(ctx, req.MarketId)
	if !found {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	var prices types.PostedPriceResponses
	for _, rp := range k.GetRawPrices(ctx, req.MarketId) {
		prices = append(prices, types.PostedPriceResponse{
			MarketID:      rp.MarketID,
			OracleAddress: rp.OracleAddress.String(),
			Price:         rp.Price,
			Expiry:        rp.Expiry,
		})
	}

	return &types.QueryRawPricesResponse{
		RawPrices: prices,
	}, nil
}
