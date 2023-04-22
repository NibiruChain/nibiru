package binding

import (
	"encoding/json"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"

	"github.com/NibiruChain/nibiru/x/common/asset"

	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"

	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
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

// CustomQuerier returns a function that is an implementation of the custom
// querier mechanism for specific messages
func CustomQuerier(qp *QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var wasmContractQuery cw_struct.BindingQuery
		if err := json.Unmarshal(request, &wasmContractQuery); err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to JSON unmarshal nibiru query: %v", err)
		}

		switch {
		case wasmContractQuery.AllMarkets != nil:
			cwResp, err := qp.Perp.AllMarkets(ctx)
			if err != nil {
				return nil, sdkerrors.Wrapf(err,
					"failed to query: perp all markets: request: %v",
					wasmContractQuery.AllMarkets)
			}
			bz, err := json.Marshal(cwResp)
			if err != nil {
				return nil, sdkerrors.Wrapf(err, ErrorMarshalResponse(cwResp))
			}
			return bz, nil
		case wasmContractQuery.Reserves != nil:
			cwResp, err := qp.Perp.Reserves(ctx, wasmContractQuery.Reserves)
			if err != nil {
				return nil, sdkerrors.Wrapf(err,
					"failed to query: perp reserves: request: %v",
					wasmContractQuery.AllMarkets)
			}
			bz, err := json.Marshal(cwResp)
			if err != nil {
				return nil, sdkerrors.Wrapf(err, ErrorMarshalResponse(cwResp))
			}
			return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.BasePrice != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.Positions != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.Position != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.PremiumFraction != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.Metrics != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.ModuleAccounts != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.PerpParams != nil:
		// 	return bz, nil
		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown nibiru query variant"}
		}
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
