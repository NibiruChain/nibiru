package keeper

import (
	"context"

	"github.com/NibiruChain/collections"

	storeprefix "github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type queryServer struct {
	k Keeper
}

func NewQuerier(k Keeper) types.QueryServer {
	return queryServer{k: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) QueryPositions(
	goCtx context.Context, req *types.QueryPositionsRequest,
) (*types.QueryPositionsResponse, error) {
	if req == nil {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, "nil request")
	}
	traderAddr, err := sdk.AccAddressFromBech32(req.Trader) // just for validation purposes
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	markets := q.k.Markets.Iterate(ctx, collections.Range[collections.Pair[asset.Pair, uint64]]{}).Values()

	var positions []types.QueryPositionResponse
	for _, market := range markets {
		amm, err := q.k.GetAMM(ctx, market.Pair)
		if err != nil {
			return nil, err
		}
		position, err := q.position(ctx, market.Pair, traderAddr, market, amm)
		if err == nil {
			positions = append(positions, position)
		}
	}

	return &types.QueryPositionsResponse{
		Positions: positions,
	}, nil
}

func (q queryServer) QueryPosition(
	goCtx context.Context, req *types.QueryPositionRequest,
) (*types.QueryPositionResponse, error) {
	if req == nil {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, "nil request")
	}
	traderAddr, err := sdk.AccAddressFromBech32(req.Trader) // just for validation purposes
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	market, err := q.k.GetMarket(ctx, req.Pair)
	if err != nil {
		return nil, err
	}

	amm, err := q.k.GetAMM(ctx, req.Pair)
	if err != nil {
		return nil, err
	}

	resp, err := q.position(ctx, req.Pair, traderAddr, market, amm)
	return &resp, err
}

func (q queryServer) QueryPositionStore(
	goCtx context.Context, req *types.QueryPositionStoreRequest,
) (resp *types.QueryPositionStoreResponse, err error) {
	if req == nil {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, "nil request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := storeprefix.NewStore(ctx.KVStore(q.k.storeKey), NamespacePositions.Prefix())

	pagination, _, err := common.ParsePagination(req.Pagination)
	if err != nil {
		return resp, grpcstatus.Error(grpccodes.InvalidArgument, err.Error())
	}

	var respPayload []types.Position
	pageRes, err := sdkquery.Paginate(store, pagination, func(key, value []byte) error {
		pos := new(types.Position)
		if err := q.k.cdc.Unmarshal(value, pos); err != nil {
			return grpcstatus.Error(grpccodes.Internal, err.Error())
		}
		respPayload = append(respPayload, *pos)
		return nil
	})
	if err != nil {
		return resp, err
	}

	return &types.QueryPositionStoreResponse{
		Positions:  respPayload,
		Pagination: pageRes,
	}, err
}

func (q queryServer) position(ctx sdk.Context, pair asset.Pair, trader sdk.AccAddress, market types.Market, amm types.AMM) (types.QueryPositionResponse, error) {
	position, err := q.k.Positions.Get(ctx, collections.Join(pair, trader))
	if err != nil {
		return types.QueryPositionResponse{}, err
	}

	positionNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return types.QueryPositionResponse{}, err
	}
	unrealizedPnl := UnrealizedPnl(position, positionNotional)

	return types.QueryPositionResponse{
		Position:         position,
		PositionNotional: positionNotional,
		UnrealizedPnl:    unrealizedPnl,
		MarginRatio:      MarginRatio(position, positionNotional, market.LatestCumulativePremiumFraction),
	}, nil
}

func (q queryServer) ModuleAccounts(
	ctx context.Context, _ *types.QueryModuleAccountsRequest,
) (*types.QueryModuleAccountsResponse, error) {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	var moduleAccountsWithBalances []types.AccountWithBalance
	for _, acc := range types.ModuleAccounts {
		account := authtypes.NewModuleAddress(acc)

		balances := q.k.BankKeeper.GetAllBalances(sdkContext, account)

		accWithBalance := types.AccountWithBalance{
			Name:    acc,
			Address: account.String(),
			Balance: balances,
		}
		moduleAccountsWithBalances = append(moduleAccountsWithBalances, accWithBalance)
	}

	return &types.QueryModuleAccountsResponse{Accounts: moduleAccountsWithBalances}, nil
}

func (q queryServer) QueryMarkets(
	goCtx context.Context, req *types.QueryMarketsRequest,
) (*types.QueryMarketsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var ammMarkets []types.AmmMarket
	markets := q.k.Markets.Iterate(ctx, collections.Range[collections.Pair[asset.Pair, uint64]]{}).Values()
	for _, market := range markets {
		// disabled markets are not returned
		if !req.Versioned && !market.Enabled {
			continue
		}

		pair := market.Pair
		amm, err := q.k.AMMs.Get(ctx, collections.Join(pair, market.Version))
		if err != nil {
			return nil, err
		}
		duo := types.AmmMarket{
			Amm:    amm,
			Market: market,
		}
		ammMarkets = append(ammMarkets, duo)
	}

	return &types.QueryMarketsResponse{AmmMarkets: ammMarkets}, nil
}
