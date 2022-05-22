package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/lockup/types"
)

// NewMsgServerImpl returns an instance of MsgServer.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		keeper: keeper,
	}
}

type msgServer struct {
	keeper Keeper
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

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k: k}
}

type queryServer struct {
	k Keeper
}

func (q queryServer) LocksByAddress(ctx context.Context, address *types.QueryLocksByAddress) (*types.QueryLocksByAddressResponse, error) {
	// TODO(mercilex): make efficient with pagination
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	addr, err := sdk.AccAddressFromBech32(address.Address)
	if err != nil {
		return nil, err
	}

	var locks []*types.Lock
	state := q.k.LocksState(sdkCtx)
	state.IterateLocksByAddress(addr, func(id uint64) (stop bool) {
		lock, err := state.Get(id)
		if err != nil {
			panic(err)
		}
		locks = append(locks, lock)
		return false
	})

	return &types.QueryLocksByAddressResponse{Locks: locks}, nil
}

func (q queryServer) LockedCoins(ctx context.Context, request *types.QueryLockedCoinsRequest) (*types.QueryLockedCoinsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	addr, err := sdk.AccAddressFromBech32(request.Address)
	if err != nil {
		return nil, err
	}
	coins := q.k.LocksState(sdkCtx).IterateLockedCoins(addr)
	return &types.QueryLockedCoinsResponse{LockedCoins: coins}, nil
}

func (q queryServer) Lock(ctx context.Context, request *types.QueryLockRequest) (*types.QueryLockResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	lock, err := q.k.LocksState(sdkCtx).Get(request.Id)
	if err != nil {
		return nil, err
	}

	return &types.QueryLockResponse{Lock: lock}, nil
}
