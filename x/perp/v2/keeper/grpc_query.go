package keeper

import (
	"context"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	traderAddr, err := sdk.AccAddressFromBech32(req.Trader) // just for validation purposes
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	markets := q.k.Markets.Iterate(ctx, collections.Range[asset.Pair]{}).Values()

	var positions []types.QueryPositionResponse
	for _, market := range markets {
		amm, err := q.k.AMMs.Get(ctx, market.Pair)
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
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	traderAddr, err := sdk.AccAddressFromBech32(req.Trader) // just for validation purposes
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	market, err := q.k.Markets.Get(ctx, req.Pair)
	if err != nil {
		return nil, err
	}

	amm, err := q.k.AMMs.Get(ctx, req.Pair)
	if err != nil {
		return nil, err
	}

	resp, err := q.position(ctx, req.Pair, traderAddr, market, amm)
	return &resp, err
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

	// marginRatioMark, err := q.k.GetMarginRatio(ctx, market, amm, position, types.MarginCalculationPriceOption_MAX_PNL)
	// if err != nil {
	// 	return nil, err
	// }

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
	goCtx context.Context, _ *types.QueryMarketsRequest,
) (*types.QueryMarketsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var ammMarkets []types.AmmMarket
	markets := q.k.Markets.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
	for _, market := range markets {
		pair := market.Pair
		amm, err := q.k.AMMs.Get(ctx, pair)
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
