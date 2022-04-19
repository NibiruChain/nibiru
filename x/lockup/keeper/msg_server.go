package keeper

import (
	"context"

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
	// TODO(heisenberg): implement
	return &types.MsgLockTokensResponse{LockId: 0}, nil
}
