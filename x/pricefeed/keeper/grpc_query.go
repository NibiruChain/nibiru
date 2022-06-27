package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) QueryPrice(goCtx context.Context, req *types.QueryPriceRequest) (*types.QueryPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pair, err := common.NewAssetPair(req.PairId)
	if err != nil {
		return nil, err
	}
	if !k.GetPairs(ctx).Contains(pair) {
		return nil, status.Error(codes.NotFound, "pair not in module params")
	}
	if !k.ActivePairsStore().getKV(ctx).Has([]byte(pair.String())) {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	tokens := common.DenomsFromPoolName(req.PairId)
	token0, token1 := tokens[0], tokens[1]
	currentPrice, sdkErr := k.GetCurrentPrice(ctx, token0, token1)
	if sdkErr != nil {
		return nil, sdkErr
	}

	return &types.QueryPriceResponse{
		Price: types.CurrentPriceResponse{PairID: req.PairId, Price: currentPrice.Price},
	}, nil
}

func (k Keeper) QueryRawPrices(
	goCtx context.Context, req *types.QueryRawPricesRequest,
) (*types.QueryRawPricesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsActivePair(ctx, req.PairId) {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	var prices types.PostedPriceResponses
	for _, rp := range k.GetRawPrices(ctx, req.PairId) {
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

func (k Keeper) QueryPrices(goCtx context.Context, req *types.QueryPricesRequest) (*types.QueryPricesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var currentPrices types.CurrentPriceResponses
	for _, currentPrice := range k.GetCurrentPrices(ctx) {
		if currentPrice.PairID != "" {
			currentPrices = append(currentPrices, types.CurrentPriceResponse(currentPrice))
		}
	}

	return &types.QueryPricesResponse{
		Prices: currentPrices,
	}, nil
}

func (k Keeper) QueryOracles(goCtx context.Context, req *types.QueryOraclesRequest) (*types.QueryOraclesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	_, err := common.NewAssetPair(req.PairId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid market ID")
	}

	oracles := k.GetOraclesForPair(ctx, req.PairId)
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

func (k Keeper) QueryParams(c context.Context, req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k Keeper) QueryMarkets(goCtx context.Context, req *types.QueryMarketsRequest,
) (*types.QueryMarketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var markets types.Markets
	for _, pair := range k.GetParams(ctx).Pairs {
		var oracleStrings []string
		for _, oracle := range k.OraclesStore().Get(ctx, pair) {
			oracleStrings = append(oracleStrings, oracle.String())
		}

		markets = append(markets, types.Market{
			PairID:  pair.String(),
			Oracles: oracleStrings,
			Active:  k.IsActivePair(ctx, pair.String()),
		})
	}

	return &types.QueryMarketsResponse{
		Markets: markets,
	}, nil
}
