package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/consensus/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// Params serves /cosmos.consensus.v1.Query/Params: the params in the
// x/consensus store, which are the ones baseapp and Nibiru BFT use. It is not
// the contract query (QueryParams), so it never runs Wasm and it reports
// Version, which the contract does not hold. Right after an x-consensus
// update in block H, it returns the new params from the end of H.
//
// The response is the same as the Cosmos SDK x/consensus query it replaces,
// so existing clients keep working.
func (k *Keeper) Params(
	goCtx context.Context, _ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	params, err := k.Get(sdk.UnwrapSDKContext(goCtx))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryParamsResponse{Params: params}, nil
}
