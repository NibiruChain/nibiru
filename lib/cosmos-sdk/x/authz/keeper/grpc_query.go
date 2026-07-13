package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz"
)

var _ authz.QueryServer = Keeper{}

// Grants implements the Query/Grants gRPC method.
// It returns grants for a granter-grantee pair. If msg type URL is set, it returns grants only for that msg type.
func (k Keeper) Grants(c context.Context, req *authz.QueryGrantsRequest) (*authz.QueryGrantsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	return &authz.QueryGrantsResponse{}, nil
}

// GranterGrants implements the Query/GranterGrants gRPC method.
func (k Keeper) GranterGrants(c context.Context, req *authz.QueryGranterGrantsRequest) (*authz.QueryGranterGrantsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	return &authz.QueryGranterGrantsResponse{}, nil
}

// GranteeGrants implements the Query/GranteeGrants gRPC method.
func (k Keeper) GranteeGrants(c context.Context, req *authz.QueryGranteeGrantsRequest) (*authz.QueryGranteeGrantsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	return &authz.QueryGranteeGrantsResponse{}, nil
}
