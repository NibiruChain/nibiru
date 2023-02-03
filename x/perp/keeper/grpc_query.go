package keeper

import (
	"context"
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
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

	pools := q.k.VpoolKeeper.GetAllPools(ctx)
	var positions []*types.QueryPositionResponse

	for _, pool := range pools {
		position, err := q.position(ctx, pool.Pair, traderAddr)
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

	return q.position(ctx, req.Pair, traderAddr)
}

func (q queryServer) position(ctx sdk.Context, pair asset.Pair, trader sdk.AccAddress) (*types.QueryPositionResponse, error) {
	position, err := q.k.Positions.Get(ctx, collections.Join(pair, trader))
	if err != nil {
		return nil, err
	}

	positionNotional, unrealizedPnl, err := q.k.getPositionNotionalAndUnrealizedPnL(ctx, position, types.PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	marginRatioMark, err := q.k.GetMarginRatio(ctx, position, types.MarginCalculationPriceOption_MAX_PNL)
	if err != nil {
		return nil, err
	}
	marginRatioIndex, err := q.k.GetMarginRatio(ctx, position, types.MarginCalculationPriceOption_INDEX)
	if err != nil {
		// The index portion of the query fails silently as not to distrupt all
		// position queries when oracles aren't posting prices.
		q.k.Logger(ctx).Error(err.Error())
		marginRatioIndex = sdk.Dec{}
	}

	return &types.QueryPositionResponse{
		Position:         &position,
		PositionNotional: positionNotional,
		UnrealizedPnl:    unrealizedPnl,
		MarginRatioMark:  marginRatioMark,
		MarginRatioIndex: marginRatioIndex,
		BlockNumber:      ctx.BlockHeight(),
	}, nil
}

func (q queryServer) Params(
	goCtx context.Context, req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: q.k.GetParams(ctx)}, nil
}

func (q queryServer) CumulativePremiumFraction(
	goCtx context.Context,
	req *types.QueryCumulativePremiumFractionRequest,
) (*types.QueryCumulativePremiumFractionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	pairMetadata, err := q.k.PairsMetadata.Get(ctx, req.Pair)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "could not find pair: %s", req.Pair)
	}

	if !q.k.VpoolKeeper.ExistsPool(ctx, pairMetadata.Pair) {
		return nil, status.Errorf(codes.NotFound, "could not find pair: %s", req.Pair)
	}

	indexTWAP, err := q.k.OracleKeeper.GetExchangeRateTwap(ctx, pairMetadata.Pair)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to fetch twap index price for pair: %s", req.Pair)
	}
	if indexTWAP.IsZero() {
		return nil, status.Errorf(codes.FailedPrecondition, "twap index price for pair: %s is zero", req.Pair)
	}

	markTwap, err := q.k.VpoolKeeper.GetMarkPriceTWAP(ctx, pairMetadata.Pair, q.k.GetParams(ctx).TwapLookbackWindow)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to fetch twap mark price for pair: %s", req.Pair)
	}
	if markTwap.IsZero() {
		return nil, status.Errorf(codes.FailedPrecondition, "twap mark price for pair: %s is zero", req.Pair)
	}

	epochInfo := q.k.EpochKeeper.GetEpochInfo(ctx, q.k.GetParams(ctx).FundingRateInterval)
	intervalsPerDay := (24 * time.Hour) / epochInfo.Duration
	premiumFraction := markTwap.Sub(indexTWAP).QuoInt64(int64(intervalsPerDay))

	return &types.QueryCumulativePremiumFractionResponse{
		CumulativePremiumFraction:              pairMetadata.LatestCumulativePremiumFraction,
		EstimatedNextCumulativePremiumFraction: pairMetadata.LatestCumulativePremiumFraction.Add(premiumFraction),
	}, nil
}

func (q queryServer) Metrics(
	goCtx context.Context, req *types.QueryMetricsRequest,
) (*types.QueryMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !q.k.VpoolKeeper.ExistsPool(ctx, req.Pair) {
		return nil, status.Errorf(codes.InvalidArgument, "pool not found: %s", req.Pair)
	}
	metrics := q.k.Metrics.GetOr(ctx, req.Pair, types.Metrics{
		Pair:        req.Pair,
		NetSize:     sdk.NewDec(0),
		VolumeQuote: sdk.NewDec(0),
		VolumeBase:  sdk.NewDec(0),
	})
	return &types.QueryMetricsResponse{Metrics: metrics}, nil
}
