package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the x/txfees keeper providing gRPC method
// handlers.
type Querier struct {
	Keeper
}

func NewQuerier(k Keeper) Querier {
	return Querier{Keeper: k}
}

func (q Querier) FeeToken(ctx context.Context, _ *types.QueryFeeTokenRequest) (*types.QueryFeeTokenResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	feeToken, err := q.Keeper.GetFeeToken(sdkCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryFeeTokenResponse{FeeToken: feeToken}, nil
}
