package keeper

import (
	"context"

	types "github.com/NibiruChain/nibiru/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the keeper with functions for gRPC queries.
type Querier struct {
	Keeper
}

func (k Keeper) Querier() Querier {
	return Querier{
		Keeper: k,
	}
}

// Params: Returns the module parameters.
func (q Querier) Params(
	goCtx context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params, _ := q.Keeper.Store.ModuleParams.Get(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// Denoms: Returns all registered denoms for a given creator.
func (q Querier) Denoms(
	goCtx context.Context,
	req *types.QueryDenomsRequest,
) (resp *types.QueryDenomsResponse, err error) {
	if req == nil {
		return resp, errNilMsg
	}
	if req.Creator == "" {
		return resp, types.ErrInvalidCreator.Wrap("empty creator address")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	iter := q.Keeper.Store.Denoms.Indexes.Creator.ExactMatch(ctx, req.Creator)
	return &types.QueryDenomsResponse{
		Denoms: iter.PrimaryKeys(),
	}, err
}

// DenomInfo: Returns all registered denoms for a given creator.
func (q Querier) DenomInfo(
	goCtx context.Context,
	req *types.QueryDenomInfoRequest,
) (resp *types.QueryDenomInfoResponse, err error) {
	if req == nil {
		return resp, errNilMsg
	}
	if err := types.DenomStr(req.Denom).Validate(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	tfMetadata, err := q.Keeper.Store.denomAdmins.Get(ctx, req.Denom)
	if err != nil {
		return resp, err
	}

	bankMetadata, _ := q.Keeper.bankKeeper.GetDenomMetaData(ctx, req.Denom)
	return &types.QueryDenomInfoResponse{
		Admin:    tfMetadata.Admin,
		Metadata: bankMetadata,
	}, err
}
