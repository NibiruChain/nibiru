package keeper

import (
	"context"
	"fmt"

	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) CreatePool(goCtx context.Context, msg *types.MsgCreatePool) (*types.MsgCreatePoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	k.Logger(ctx).Info(fmt.Sprintf("Sender address is %s", sender))

	_, err = k.NewPool(ctx, sender, *msg.PoolParams, msg.PoolAssets)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreatePoolResponse{}, nil
}
