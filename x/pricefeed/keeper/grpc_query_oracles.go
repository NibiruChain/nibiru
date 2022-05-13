package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

func (k Keeper) Oracles(goCtx context.Context, req *types.QueryOraclesRequest) (*types.QueryOraclesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	oracles, err := k.GetOracles(ctx, req.PairId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	var strOracles []string
	for _, oracle := range oracles {
		strOracles = append(strOracles, oracle.String())
	}

	return &types.QueryOraclesResponse{
		Oracles: strOracles,
	}, nil
}
