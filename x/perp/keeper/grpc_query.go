package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

type queryServer struct {
	Keeper
}

func NewQuerier(k Keeper) queryServer {
	return queryServer{Keeper: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) TraderPosition(
	goCtx context.Context, req *types.QueryTraderPositionRequest,
) (*types.QueryTraderPositionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	trader, err := sdk.AccAddressFromBech32(req.Trader)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pair, err := common.NewAssetPair(req.TokenPair)
	if err != nil {
		return nil, err
	}

	position, err := q.Keeper.PositionsState(ctx).Get(pair, trader)
	if err != nil {
		return nil, err
	}

	return &types.QueryTraderPositionResponse{
		Position: position,
	}, nil
}
