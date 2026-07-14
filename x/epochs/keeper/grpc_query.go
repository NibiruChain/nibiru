package keeper

import (
	"context"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/epochs"
)

var _ epochs.QueryServer = Querier{}

// Querier defines a wrapper around the x/epochs keeper providing gRPC method
// handlers.
type Querier struct {
	Keeper
}

func NewQuerier(k Keeper) Querier {
	return Querier{Keeper: k}
}

// EpochInfos provide running epochInfos.
func (q Querier) EpochInfos(c context.Context, _ *epochs.QueryEpochInfosRequest) (*epochs.QueryEpochInfosResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	return &epochs.QueryEpochInfosResponse{
		Epochs: q.AllEpochInfos(ctx),
	}, nil
}

// CurrentEpoch provides current epoch of specified identifier.
func (q Querier) CurrentEpoch(c context.Context, req *epochs.QueryCurrentEpochRequest) (resp *epochs.QueryCurrentEpochResponse, err error) {
	ctx := sdk.UnwrapSDKContext(c)

	info, err := q.GetEpochInfo(ctx, req.Identifier)
	if err != nil {
		return
	}

	return &epochs.QueryCurrentEpochResponse{
		CurrentEpoch: info.CurrentEpoch,
	}, nil
}
