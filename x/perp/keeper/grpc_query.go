package keeper

import (
	"context"
	"fmt"

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

	fmt.Println("STEVENDEBUG TraderPosition new: ")

	ctx := sdk.UnwrapSDKContext(goCtx)
	position, err := q.Keeper.Positions().Get(ctx, common.TokenPair(req.TokenPair), req.Trader)

	fmt.Println("STEVENDEBUG position: ", position)
	fmt.Println("STEVENDEBUG err: ", err)

	return &types.QueryTraderPositionResponse{
		Position: position,
	}, nil
}

func (q queryServer) TraderMargin(
	goCtx context.Context, req *types.QueryTraderMarginRequest,
) (*types.QueryTraderMarginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	fmt.Println("STEVENDEBUG TraderMargin new: ")

	// TODO:
	// ctx := sdk.UnwrapSDKContext(goCtx)
	// var balances sdk.Coins = k.BankKeeper.GetAllBalances(
	// 	ctx, k.AccountKeeper.GetModuleAddress(types.ModuleName),
	// )

	return &types.QueryTraderMarginResponse{}, nil
}
