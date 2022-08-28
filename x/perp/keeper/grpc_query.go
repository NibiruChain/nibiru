package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

type queryServer struct {
	Keeper
}

func NewQuerier(k Keeper) queryServer {
	return queryServer{Keeper: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) QueryTraderPosition(
	goCtx context.Context, req *types.QueryTraderPositionRequest,
) (*types.QueryTraderPositionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	trader, err := sdk.AccAddressFromBech32(req.Trader)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pair, err := common.NewAssetPair(req.TokenPair)
	if err != nil {
		return nil, err
	}

	position, err := q.Keeper.PositionsState(ctx).Get(pair, trader)
	if err != nil {
		return nil, err
	}

	positionNotional, unrealizedPnl, err := q.Keeper.getPositionNotionalAndUnrealizedPnL(ctx, *position, types.PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	marginRatioMark, err := q.Keeper.GetMarginRatio(ctx, *position, types.MarginCalculationPriceOption_MAX_PNL)
	if err != nil {
		return nil, err
	}
	marginRatioIndex, err := q.Keeper.GetMarginRatio(ctx, *position, types.MarginCalculationPriceOption_INDEX)
	if err != nil {
		// The index portion of the query fails silently as not to distrupt all
		// position queries when oracles aren't posting prices.
		q.Keeper.Logger(ctx).Error(err.Error())
		marginRatioIndex = sdk.Dec{}
	}

	return &types.QueryTraderPositionResponse{
		Position:         position,
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

	return &types.QueryParamsResponse{Params: q.Keeper.GetParams(ctx)}, nil
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

	pairMetadata, err := q.Keeper.PairMetadataState(ctx).Get(assetPair)
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
