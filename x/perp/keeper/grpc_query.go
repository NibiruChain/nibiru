package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"
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

	pair, err := common.NewAssetPair(req.TokenPair)
	if err != nil {
		return nil, err
	}

	return q.position(ctx, pair, traderAddr)
}

func (q queryServer) position(ctx sdk.Context, pair common.AssetPair, trader sdk.AccAddress) (*types.QueryPositionResponse, error) {
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

func (q queryServer) FundingRates(
	goCtx context.Context, req *types.QueryFundingRatesRequest,
) (*types.QueryFundingRatesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	assetPair, err := common.NewAssetPair(req.Pair)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid pair: %s", req.Pair)
	}

	pairMetadata, err := q.k.PairsMetadata.Get(ctx, assetPair)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "could not find pair: %s", req.Pair)
	}

	var fundingRates []sdk.Dec

	// truncate to most recent 48 funding payments
	// given 30 minute funding rate calculations, this should give the last 24 hours of funding payments
	numFundingRates := len(pairMetadata.CumulativePremiumFractions)
	if numFundingRates >= 48 {
		fundingRates = pairMetadata.CumulativePremiumFractions[numFundingRates-48 : numFundingRates]
	} else {
		fundingRates = pairMetadata.CumulativePremiumFractions
	}

	return &types.QueryFundingRatesResponse{
		CumulativeFundingRates: fundingRates,
	}, nil
}
