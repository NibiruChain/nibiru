package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/lockup/types"
)

type msgServer struct {
	keeper *LockupKeeper
}

// NewMsgServerImpl returns an instance of MsgServer.
func NewMsgServerImpl(keeper *LockupKeeper) types.MsgServer {
	return &msgServer{
		keeper: keeper,
	}
}

var _ types.MsgServer = msgServer{}

func (server msgServer) LockTokens(goCtx context.Context, msg *types.MsgLockTokens) (*types.MsgLockTokensResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, err
	}

	lockID, err := server.keeper.LockTokens(ctx, addr, msg.Coins, msg.Duration)
	if err != nil {
		return nil, err
	}

	return &types.MsgLockTokensResponse{LockId: lockID.LockId}, nil
}

func (server msgServer) InitiateUnlock(ctx context.Context, unlock *types.MsgInitiateUnlock) (*types.MsgInitiateUnlockResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, err := server.keeper.UnlockTokens(sdkCtx, unlock.LockId)
	return &types.MsgInitiateUnlockResponse{}, err
}
