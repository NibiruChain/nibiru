package binding

import (
	"encoding/json"
	"errors"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/v1/amm/keeper"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/v1/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/v1/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

type QueryPlugin struct {
	Perp *PerpQuerier
}

// NewQueryPlugin returns a pointer to a new QueryPlugin
func NewQueryPlugin(perp perpkeeper.Keeper, perpAmm perpammkeeper.Keeper) QueryPlugin {
	return QueryPlugin{
		Perp: &PerpQuerier{
			perp:    perpkeeper.NewQuerier(perp),
			perpAmm: perpammkeeper.NewQuerier(perpAmm),
		},
	}
}

func (qp *QueryPlugin) ToBinary(
	cwResp any, err error, cwReq any,
) ([]byte, error) {
	if err != nil {
		return nil, sdkerrors.Wrapf(err,
			"failed to query: perp all markets: request: %v",
			cwReq)
	}
	bz, err := json.Marshal(cwResp)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "failed to JSON marshal response: %v", cwResp)
	}
	return bz, nil
}

// CustomQuerier returns a function that is an implementation of the custom
// querier mechanism for specific messages
func CustomQuerier(qp QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var wasmContractQuery cw_struct.BindingQuery
		if err := json.Unmarshal(request, &wasmContractQuery); err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to JSON unmarshal nibiru query: %v", err)
		}

		switch {
		case wasmContractQuery.AllMarkets != nil:
			cwReq := wasmContractQuery.AllMarkets
			cwResp, err := qp.Perp.AllMarkets(ctx)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.Reserves != nil:
			cwReq := wasmContractQuery.Reserves
			cwResp, err := qp.Perp.Reserves(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.BasePrice != nil:
			cwReq := wasmContractQuery.BasePrice
			cwResp, err := qp.Perp.BasePrice(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.Positions != nil:
			cwReq := wasmContractQuery.Positions
			cwResp, err := qp.Perp.Positions(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.Position != nil:
			cwReq := wasmContractQuery.Position
			cwResp, err := qp.Perp.Position(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.PremiumFraction != nil:
			cwReq := wasmContractQuery.PremiumFraction
			cwResp, err := qp.Perp.PremiumFraction(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.Metrics != nil:
			cwReq := wasmContractQuery.Metrics
			cwResp, err := qp.Perp.Metrics(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.ModuleAccounts != nil:
			cwReq := wasmContractQuery.ModuleAccounts
			cwResp, err := qp.Perp.ModuleAccounts(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		case wasmContractQuery.PerpParams != nil:
			cwReq := wasmContractQuery.PerpParams
			cwResp, err := qp.Perp.ModuleParams(ctx, cwReq)
			return qp.ToBinary(cwResp, err, cwReq)

		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown nibiru query variant"}
		}
	}
}

// ----------------------------------------------------------------------
// PerpQuerier
// ----------------------------------------------------------------------

type PerpQuerier struct {
	perp    perptypes.QueryServer
	perpAmm perpammtypes.QueryServer
}

func (perpExt *PerpQuerier) Reserves(
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
		BaseReserve:  sdkResp.BaseReserve,
		QuoteReserve: sdkResp.QuoteReserve,
	}, err
}

func (perpExt *PerpQuerier) AllMarkets(
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
			BaseReserve:  pbMarket.BaseReserve,
			QuoteReserve: pbMarket.QuoteReserve,
			SqrtDepth:    pbMarket.SqrtDepth,
			Depth:        pbPrice.SwapInvariant,
			TotalLong:    pbMarket.TotalLong,
			TotalShort:   pbMarket.TotalShort,
			PegMult:      pbMarket.PegMultiplier,
			Config: &cw_struct.MarketConfig{
				TradeLimitRatio:        pbMarket.Config.TradeLimitRatio,
				FluctLimitRatio:        pbMarket.Config.FluctuationLimitRatio,
				MaxOracleSpreadRatio:   pbMarket.Config.MaxOracleSpreadRatio,
				MaintenanceMarginRatio: pbMarket.Config.MaintenanceMarginRatio,
				MaxLeverage:            pbMarket.Config.MaxLeverage,
			},
			MarkPrice:   pbPrice.MarkPrice,
			IndexPrice:  pbPrice.IndexPrice,
			TwapMark:    pbPrice.TwapMark,
			BlockNumber: sdk.NewInt(ctx.BlockHeight()),
		}
	}

	return &cw_struct.AllMarketsResponse{
		MarketMap: marketMap,
	}, err
}

func (perpExt *PerpQuerier) BasePrice(
	ctx sdk.Context, cwReq *cw_struct.BasePriceRequest,
) (*cw_struct.BasePriceResponse, error) {
	pair, err := asset.TryNewPair(cwReq.Pair)
	if err != nil {
		return nil, err
	}

	var direction perpammtypes.Direction
	if cwReq.IsLong {
		direction = perpammtypes.Direction_LONG
	} else {
		direction = perpammtypes.Direction_SHORT
	}

	sdkReq := &perpammtypes.QueryBaseAssetPriceRequest{
		Pair:            pair,
		Direction:       direction,
		BaseAssetAmount: cwReq.BaseAmount.ToDec(),
	}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perpAmm.BaseAssetPrice(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}
	return &cw_struct.BasePriceResponse{
		Pair:        pair.String(),
		BaseAmount:  cwReq.BaseAmount.ToDec(),
		QuoteAmount: sdkResp.PriceInQuoteDenom,
		IsLong:      cwReq.IsLong,
	}, err
}

func (perpExt *PerpQuerier) PremiumFraction(
	ctx sdk.Context, cwReq *cw_struct.PremiumFractionRequest,
) (*cw_struct.PremiumFractionResponse, error) {
	pair, err := asset.TryNewPair(cwReq.Pair)
	if err != nil {
		return nil, err
	}

	sdkReq := &perptypes.QueryCumulativePremiumFractionRequest{
		Pair: pair,
	}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perp.CumulativePremiumFraction(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}

	return &cw_struct.PremiumFractionResponse{
		Pair:             pair.String(),
		CPF:              sdkResp.CumulativePremiumFraction,
		EstimatedNextCPF: sdkResp.EstimatedNextCumulativePremiumFraction,
	}, err
}

func (perpExt *PerpQuerier) Metrics(
	ctx sdk.Context, cwReq *cw_struct.MetricsRequest,
) (*cw_struct.MetricsResponse, error) {
	pair, err := asset.TryNewPair(cwReq.Pair)
	if err != nil {
		return nil, err
	}

	sdkReq := &perptypes.QueryMetricsRequest{
		Pair: pair,
	}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perp.Metrics(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}

	return &cw_struct.MetricsResponse{
		Metrics: cw_struct.Metrics{
			Pair:        sdkResp.Metrics.Pair.String(),
			NetSize:     sdkResp.Metrics.NetSize,
			VolumeQuote: sdkResp.Metrics.VolumeQuote,
			VolumeBase:  sdkResp.Metrics.VolumeBase,
			BlockNumber: sdk.NewInt(ctx.BlockHeight()),
		},
	}, err
}

func (perpExt *PerpQuerier) ModuleAccounts(
	ctx sdk.Context, cwReq *cw_struct.ModuleAccountsRequest,
) (*cw_struct.ModuleAccountsResponse, error) {
	if cwReq == nil {
		return nil, errors.New("nil request")
	}
	sdkReq := &perptypes.QueryModuleAccountsRequest{}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perp.ModuleAccounts(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}

	moduleAccounts := make(map[string]cw_struct.ModuleAccountWithBalance)
	for _, acc := range sdkResp.Accounts {
		addr, err := sdk.AccAddressFromBech32(acc.Address)
		if err != nil {
			return nil, err
		}
		moduleAccounts[acc.Name] = cw_struct.ModuleAccountWithBalance{
			Name:    acc.Name,
			Addr:    addr,
			Balance: acc.Balance,
		}
	}

	return &cw_struct.ModuleAccountsResponse{
		ModuleAccounts: moduleAccounts,
	}, err
}

func (perpExt *PerpQuerier) ModuleParams(
	ctx sdk.Context, cwReq *cw_struct.PerpParamsRequest,
) (*cw_struct.PerpParamsResponse, error) {
	if cwReq == nil {
		return nil, errors.New("nil request")
	}
	sdkReq := &perptypes.QueryParamsRequest{}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perp.Params(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}

	params := sdkResp.Params

	lookback := sdk.NewInt(int64(params.TwapLookbackWindow.Seconds()))

	liquidators := []string{}
	liquidators = append(liquidators, params.WhitelistedLiquidators...)

	return &cw_struct.PerpParamsResponse{
		ModuleParams: cw_struct.PerpParams{
			Stopped:                 params.Stopped,
			FeePoolFeeRatio:         params.FeePoolFeeRatio,
			EcosystemFundFeeRatio:   params.EcosystemFundFeeRatio,
			LiquidationFeeRatio:     params.LiquidationFeeRatio,
			PartialLiquidationRatio: params.PartialLiquidationRatio,
			FundingRateInterval:     params.FundingRateInterval,
			TwapLookbackWindow:      lookback,
			WhitelistedLiquidators:  liquidators,
		},
	}, err
}

func (perpExt *PerpQuerier) Position(
	ctx sdk.Context, cwReq *cw_struct.PositionRequest,
) (*cw_struct.PositionResponse, error) {
	pair, err := asset.TryNewPair(cwReq.Pair)
	if err != nil {
		return nil, err
	}
	sdkReq := &perptypes.QueryPositionRequest{
		Pair:   pair,
		Trader: cwReq.Trader,
	}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perp.QueryPosition(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}
	return &cw_struct.PositionResponse{
		Position: cw_struct.Position{
			TraderAddr:   sdkResp.Position.TraderAddress,
			Pair:         sdkResp.Position.Pair.String(),
			Size:         sdkResp.Position.Size_,
			Margin:       sdkResp.Position.Margin,
			OpenNotional: sdkResp.Position.OpenNotional,
			LatestCPF:    sdkResp.Position.LatestCumulativePremiumFraction,
			BlockNumber:  sdk.NewInt(sdkResp.Position.BlockNumber)},
		Notional:           sdkResp.PositionNotional,
		Upnl:               sdkResp.UnrealizedPnl,
		Margin_ratio_mark:  sdkResp.MarginRatioMark,
		Margin_ratio_index: sdkResp.MarginRatioIndex,
		Block_number:       sdk.NewInt(sdkResp.BlockNumber),
	}, err
}

func (perpExt *PerpQuerier) Positions(
	ctx sdk.Context, cwReq *cw_struct.PositionsRequest,
) (*cw_struct.PositionsResponse, error) {
	sdkReq := &perptypes.QueryPositionsRequest{
		Trader: cwReq.Trader,
	}
	goCtx := sdk.WrapSDKContext(ctx)
	sdkResp, err := perpExt.perp.QueryPositions(goCtx, sdkReq)
	if err != nil {
		return nil, err
	}

	positionMap := make(map[string]cw_struct.Position)
	for _, posResp := range sdkResp.Positions {
		pair := posResp.Position.Pair.String()
		pos := posResp.Position
		positionMap[pair] = cw_struct.Position{
			TraderAddr:   pos.TraderAddress,
			Pair:         pair,
			Size:         pos.Size_,
			Margin:       pos.Margin,
			OpenNotional: pos.OpenNotional,
			LatestCPF:    pos.LatestCumulativePremiumFraction,
			BlockNumber:  sdk.NewInt(pos.BlockNumber),
		}
	}

	return &cw_struct.PositionsResponse{
		Positions: positionMap,
	}, err
}
