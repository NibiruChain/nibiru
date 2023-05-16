package keeper

import (
	"context"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type queryServer struct {
	k Keeper
}

func NewQuerier(k Keeper) v2types.QueryServer {
	return queryServer{k: k}
}

var _ v2types.QueryServer = queryServer{}

func (q queryServer) QueryPositions(
	goCtx context.Context, req *v2types.QueryPositionsRequest,
) (*v2types.QueryPositionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	traderAddr, err := sdk.AccAddressFromBech32(req.Trader) // just for validation purposes
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	markets := q.k.Markets.Iterate(ctx, collections.Range[asset.Pair]{}).Values()

	var positions []v2types.QueryPositionResponse
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

	return &v2types.QueryPositionsResponse{
		Positions: positions,
	}, nil
}

func (q queryServer) QueryPosition(
	goCtx context.Context, req *v2types.QueryPositionRequest,
) (*v2types.QueryPositionResponse, error) {
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

func (q queryServer) position(ctx sdk.Context, pair asset.Pair, trader sdk.AccAddress, market v2types.Market, amm v2types.AMM) (v2types.QueryPositionResponse, error) {
	position, err := q.k.Positions.Get(ctx, collections.Join(pair, trader))
	if err != nil {
		return v2types.QueryPositionResponse{}, err
	}

	positionNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return v2types.QueryPositionResponse{}, err
	}
	unrealizedPnl := UnrealizedPnl(position, positionNotional)

	// marginRatioMark, err := q.k.GetMarginRatio(ctx, market, amm, position, types.MarginCalculationPriceOption_MAX_PNL)
	// if err != nil {
	// 	return nil, err
	// }

	return v2types.QueryPositionResponse{
		Position:         position,
		PositionNotional: positionNotional,
		UnrealizedPnl:    unrealizedPnl,
		MarginRatio:      MarginRatio(position, positionNotional, market.LatestCumulativePremiumFraction),
	}, nil
}

func (q queryServer) ModuleAccounts(
	ctx context.Context, _ *v2types.QueryModuleAccountsRequest,
) (*v2types.QueryModuleAccountsResponse, error) {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	var moduleAccountsWithBalances []v2types.AccountWithBalance
	for _, acc := range types.ModuleAccounts {
		account := authtypes.NewModuleAddress(acc)

		balances := q.k.BankKeeper.GetAllBalances(sdkContext, account)

		accWithBalance := v2types.AccountWithBalance{
			Name:    acc,
			Address: account.String(),
			Balance: balances,
		}
		moduleAccountsWithBalances = append(moduleAccountsWithBalances, accWithBalance)
	}

	return &v2types.QueryModuleAccountsResponse{Accounts: moduleAccountsWithBalances}, nil
}

func (q queryServer) QueryMarkets(
	goCtx context.Context, _ *v2types.QueryMarketsRequest,
) (*v2types.QueryMarketsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var duos []v2types.AmmMarketDuo
	markets := q.k.Markets.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
	for _, market := range markets {
		pair := market.Pair
		amm, err := q.k.AMMs.Get(ctx, pair)
		if err != nil {
			return nil, err
		}
		duo := v2types.AmmMarketDuo{
			Amm:    amm,
			Market: market,
		}
		duos = append(duos, duo)
	}

	return &v2types.QueryMarketsResponse{Duos: duos}, nil
}
