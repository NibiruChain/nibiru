package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

type queryServer struct {
	k Keeper
}

func NewQuerier(k Keeper) types.QueryServer {
	return queryServer{k: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) QueryPrice(goCtx context.Context, req *types.QueryPriceRequest) (*types.QueryPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pair, err := common.NewAssetPair(req.PairId)
	if err != nil {
		return nil, err
	}
	if !q.k.GetPairs(ctx).Contains(pair) {
		return nil, status.Error(codes.NotFound, "pair not in module params")
	}
	if !q.k.ActivePairsStore().getKV(ctx).Has([]byte(pair.String())) {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	tokens := strings.Split(req.PairId, common.PairSeparator)
	token0, token1 := tokens[0], tokens[1]
	currentPrice, err := q.k.GetCurrentPrice(ctx, token0, token1)
	if err != nil {
		return nil, err
	}

	twap, err := q.k.GetCurrentTWAP(ctx, token0, token1)
	if err != nil {
		return nil, err
	}

	return &types.QueryPriceResponse{
		Price: types.CurrentPriceResponse{
			PairID: req.PairId,
			Price:  currentPrice.Price,
			Twap:   twap,
		},
	}, nil
}

func (q queryServer) QueryRawPrices(
	goCtx context.Context, req *types.QueryRawPricesRequest,
) (*types.QueryRawPricesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !q.k.IsActivePair(ctx, req.PairId) {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	var prices types.PostedPriceResponses
	for _, rp := range q.k.GetRawPrices(ctx, req.PairId) {
		prices = append(prices, types.PostedPriceResponse{
			PairID:        rp.PairID,
			OracleAddress: rp.Oracle,
			Price:         rp.Price,
			Expiry:        rp.Expiry,
		})
	}

	return &types.QueryRawPricesResponse{
		RawPrices: prices,
	}, nil
}

func (q queryServer) QueryPrices(goCtx context.Context, req *types.QueryPricesRequest) (*types.QueryPricesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var currentPrices types.CurrentPriceResponses
	for _, currentPrice := range q.k.GetCurrentPrices(ctx) {
		if currentPrice.PairID != "" {
			currentPrices = append(currentPrices, types.CurrentPriceResponse{
				PairID: currentPrice.PairID,
				Price:  currentPrice.Price,
			})
		}
	}

	return &types.QueryPricesResponse{
		Prices: currentPrices,
	}, nil
}

func (q queryServer) QueryOracles(goCtx context.Context, req *types.QueryOraclesRequest) (*types.QueryOraclesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	_, err := common.NewAssetPair(req.PairId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	oracles := q.k.GetOraclesForPair(ctx, req.PairId)
	if len(oracles) == 0 {
		return &types.QueryOraclesResponse{}, nil
	}

	var strOracles []string
	for _, oracle := range oracles {
		strOracles = append(strOracles, oracle.String())
	}

	return &types.QueryOraclesResponse{
		Oracles: strOracles,
	}, nil
}

func (q queryServer) QueryParams(c context.Context, req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryParamsResponse{Params: q.k.GetParams(ctx)}, nil
}

func (q queryServer) QueryMarkets(goCtx context.Context, req *types.QueryMarketsRequest,
) (*types.QueryMarketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var markets types.Markets
	for _, pair := range q.k.GetParams(ctx).Pairs {
		var oracleStrings []string
		for _, oracle := range q.k.OraclesStore().Get(ctx, pair) {
			oracleStrings = append(oracleStrings, oracle.String())
		}

		markets = append(markets, types.Market{
			PairID:  pair.String(),
			Oracles: oracleStrings,
			Active:  q.k.IsActivePair(ctx, pair.String()),
		})
	}

	return &types.QueryMarketsResponse{
		Markets: markets,
	}, nil
}
