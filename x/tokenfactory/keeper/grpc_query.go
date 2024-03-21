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

// QueryDenoms: Returns all registered denoms for a given creator.
func (k Keeper) QueryDenoms(ctx sdk.Context, creator string) []string {
	iter := k.Store.Denoms.Indexes.Creator.ExactMatch(ctx, creator)
	return iter.PrimaryKeys()
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
	return &types.QueryDenomsResponse{
		Denoms: q.Keeper.QueryDenoms(ctx, req.Creator),
	}, err
}

// QueryDenomInfo: Returns bank and tokenfactory metadata for a registered denom.
func (k Keeper) QueryDenomInfo(
	ctx sdk.Context, denom string,
) (resp *types.QueryDenomInfoResponse, err error) {
	tfMetadata, err := k.Store.denomAdmins.Get(ctx, denom)
	if err != nil {
		return resp, err
	}

	bankMetadata, _ := k.BankKeeper.GetDenomMetaData(ctx, denom)
	return &types.QueryDenomInfoResponse{
		Admin:    tfMetadata.Admin,
		Metadata: bankMetadata,
	}, err
}

// DenomInfo: Returns bank and tokenfactory metadata for a registered denom.
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
	return q.Keeper.QueryDenomInfo(ctx, req.Denom)
}
