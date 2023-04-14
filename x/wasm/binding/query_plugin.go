package binding

import (
	"github.com/NibiruChain/nibiru/x/common/asset"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"

	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type QueryPlugin struct {
	Perp *PerpExtension
}

// NewQueryPlugin returns a pointer to a new QueryPlugin
func NewQueryPlugin(perp *perpkeeper.Keeper, perpAmm *perpammkeeper.Keeper) *QueryPlugin {
	return &QueryPlugin{
		Perp: &PerpExtension{
			perp:    perpkeeper.NewQuerier(*perp),
			perpAmm: perpammkeeper.NewQuerier(*perpAmm),
		},
	}
}

// ----------------------------------------------------------------------
// PerpExtension
// ----------------------------------------------------------------------

type PerpExtension struct {
	perp    perptypes.QueryServer
	perpAmm perpammtypes.QueryServer
}

func (perpExt *PerpExtension) Reserves(
	ctx sdk.Context, cwReq *cw_struct.ReservesRequest,
) (*cw_struct.ReservesResponse, error) {
	pair := asset.Pair(cwReq.Pair)
	sdkReq := &perpammtypes.QueryReserveAssetsRequest{
		Pair: pair,
	}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perpAmm.ReserveAssets(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}
	return &cw_struct.ReservesResponse{
		Pair:         pair.String(),
		BaseReserve:  sdkResp.BaseAssetReserve,
		QuoteReserve: sdkResp.QuoteAssetReserve,
	}, err
}

func (perpExt *PerpExtension) AllMarkets(
	ctx sdk.Context,
) (*cw_struct.AllMarketsResponse, error) {
	sdkReq := &perpammtypes.QueryAllPoolsRequest{}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perpAmm.AllPools(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}

	marketMap := make(map[string]cw_struct.Market)
	for idx, pbMarket := range sdkResp.Markets {
		pbPrice := sdkResp.Prices[idx]
		key := pbMarket.Pair.String()
		marketMap[key] = cw_struct.Market{
			Pair:         key,
			BaseReserve:  pbMarket.BaseAssetReserve,
			QuoteReserve: pbMarket.QuoteAssetReserve,
			SqrtDepth:    pbMarket.SqrtDepth,
			Depth:        pbPrice.SwapInvariant,
			Bias:         pbMarket.Bias,
			PegMult:      pbMarket.PegMultiplier,
			Config: cw_struct.MarketConfig{
				TradeLimitRatio:        pbMarket.Config.TradeLimitRatio,
				FluctLimitRatio:        pbMarket.Config.FluctuationLimitRatio,
				MaxOracleSpreadRatio:   pbMarket.Config.MaxOracleSpreadRatio,
				MaintenanceMarginRatio: pbMarket.Config.MaintenanceMarginRatio,
				MaxLeverage:            pbMarket.Config.MaxLeverage,
			},
			MarkPrice:   pbPrice.MarkPrice,
			IndexPrice:  pbPrice.IndexPrice,
			TwapMark:    pbPrice.TwapMark,
			BlockNumber: ctx.BlockHeight(),
		}
	}

	return &cw_struct.AllMarketsResponse{
		MarketMap: marketMap,
	}, err
}
