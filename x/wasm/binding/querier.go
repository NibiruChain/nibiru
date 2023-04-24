package binding

import (
	"encoding/json"
	"errors"
	"fmt"

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
		return nil, sdkerrors.Wrapf(err, ErrorMarshalResponse(cwResp))
	}
	return bz, nil
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

		// TODO implement
		// TODO test
		// case wasmContractQuery.Positions != nil:
		// 	return bz, nil
		// TODO implement
		// TODO test
		// case wasmContractQuery.Position != nil:
		// 	return bz, nil

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

		// TODO test
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

func (perpExt *PerpExtension) BasePrice(
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

func (perpExt *PerpExtension) PremiumFraction(
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

func (perpExt *PerpExtension) Metrics(
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

func (perpExt *PerpExtension) ModuleAccounts(
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

func (perpExt *PerpExtension) ModuleParams(
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

	fmt.Printf("\nDEBUG-UD params lookback: %v", params.TwapLookbackWindow)
	lookback := sdk.NewInt(int64(params.TwapLookbackWindow.Seconds()))
	fmt.Printf("\nDEBUG-UD lookback: %v", lookback)

	liquidators := []string{}
	for _, liq := range params.WhitelistedLiquidators {
		liquidators = append(liquidators, liq)
	}
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
