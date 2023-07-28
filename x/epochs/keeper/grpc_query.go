package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the x/epochs keeper providing gRPC method
// handlers.
type Querier struct {
	Keeper
}

func NewQuerier(k Keeper) Querier {
	return Querier{Keeper: k}
}

// EpochInfos provide running epochInfos.
func (q Querier) EpochInfos(c context.Context, _ *types.QueryEpochsInfoRequest) (*types.QueryEpochsInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryEpochsInfoResponse{
		Epochs: q.Keeper.AllEpochInfos(ctx),
	}, nil
}

// CurrentEpoch provides current epoch of specified identifier.
func (q Querier) CurrentEpoch(c context.Context, req *types.QueryCurrentEpochRequest) (resp *types.QueryCurrentEpochResponse, err error) {
	ctx := sdk.UnwrapSDKContext(c)

	info, err := q.Keeper.GetEpochInfo(ctx, req.Identifier)
	if err != nil {
		return
	}

	return &types.QueryCurrentEpochResponse{
		CurrentEpoch: info.CurrentEpoch,
	}, nil
}
