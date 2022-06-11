package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

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

func (s msgServer) Unlock(ctx context.Context, unlock *types.MsgUnlock) (*types.MsgUnlockResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, err := s.keeper.UnlockTokens(sdkCtx, unlock.LockId)
	if err != nil {
		return nil, err
	}

	return &types.MsgUnlockResponse{}, nil
}

var _ types.MsgServer = msgServer{}

func (s msgServer) LockTokens(goCtx context.Context, msg *types.MsgLockTokens) (*types.MsgLockTokensResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, err
	}

	lockID, err := s.keeper.LockTokens(ctx, addr, msg.Coins, msg.Duration)
	if err != nil {
		return nil, err
	}

	return &types.MsgLockTokensResponse{LockId: lockID.LockId}, nil
}

func (s msgServer) InitiateUnlock(ctx context.Context, unlock *types.MsgInitiateUnlock) (*types.MsgInitiateUnlockResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, err := s.keeper.InitiateUnlocking(sdkCtx, unlock.LockId)
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

	state := q.k.LocksState(sdkCtx)

	key := state.keyAddr(addr.String(), nil)
	store := prefix.NewStore(state.addrIndex, key)

	var locks []*types.Lock
	res, err := query.Paginate(store, address.Pagination, func(key []byte, _ []byte) error {
		value := state.locks.Get(key)
		if value == nil {
			panic(fmt.Errorf("state corruption cannot find key: %x", key))
		}
		lock := new(types.Lock)
		q.k.cdc.MustUnmarshal(value, lock)
		locks = append(locks, lock)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryLocksByAddressResponse{Locks: locks, Pagination: res}, nil
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
